/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	csiclient "k8s.io/csi-api/pkg/client/clientset/versioned"
	csiinformers "k8s.io/csi-api/pkg/client/informers/externalversions"
	"k8s.io/klog"

	"github.com/kubernetes-csi/external-attacher/pkg/connection"
	"github.com/kubernetes-csi/external-attacher/pkg/controller"
)

const (
	// Number of worker threads
	threads = 10

	// Default timeout of short CSI calls like GetPluginInfo
	csiTimeout = time.Second

	// Name of CSI plugin for dummy operation
	dummyAttacherName = "csi/dummy"
)

// Command line flags
var (
	kubeconfig        = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Required only when running out of cluster.")
	resync            = flag.Duration("resync", 10*time.Minute, "Resync interval of the controller.")
	connectionTimeout = flag.Duration("connection-timeout", 0, "This option is deprecated.")
	csiAddress        = flag.String("csi-address", "/run/csi/socket", "Address of the CSI driver socket.")
	dummy             = flag.Bool("dummy", false, "Run in dummy mode, i.e. not connecting to CSI driver and marking everything as attached. Expected CSI driver name is \"csi/dummy\".")
	showVersion       = flag.Bool("version", false, "Show version.")
	timeout           = flag.Duration("timeout", 15*time.Second, "Timeout for waiting for attaching or detaching the volume.")

	enableLeaderElection    = flag.Bool("leader-election", false, "Enable leader election.")
	leaderElectionNamespace = flag.String("leader-election-namespace", "", "Namespace where this attacher runs.")
	leaderElectionIdentity  = flag.String("leader-election-identity", "", "Unique identity of this attacher. Typically name of the pod where the attacher runs.")
)

var (
	version = "unknown"
)

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Parse()

	if *showVersion {
		fmt.Println(os.Args[0], version)
		return
	}
	klog.Infof("Version: %s", version)

	if *connectionTimeout != 0 {
		klog.Warningf("Warning: option -connection-timeout is deprecated and has no effect")
	}

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	csiClientset, err := csiclient.NewForConfig(config)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	factory := informers.NewSharedInformerFactory(clientset, *resync)
	var csiFactory csiinformers.SharedInformerFactory
	var handler controller.Handler

	var attacher string
	if *dummy {
		// Do not connect to any CSI, mark everything as attached.
		handler = controller.NewTrivialHandler(clientset)
		attacher = dummyAttacherName
	} else {
		// Connect to CSI.
		csiConn, err := connection.New(*csiAddress)
		if err != nil {
			klog.Error(err.Error())
			os.Exit(1)
		}

		err = csiConn.Probe(*timeout)
		if err != nil {
			klog.Error(err.Error())
			os.Exit(1)
		}

		// Find driver name.
		ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
		defer cancel()
		attacher, err = csiConn.GetDriverName(ctx)
		if err != nil {
			klog.Error(err.Error())
			os.Exit(1)
		}
		klog.V(2).Infof("CSI driver name: %q", attacher)

		supportsService, err := csiConn.SupportsPluginControllerService(ctx)
		if err != nil {
			klog.Error(err.Error())
			os.Exit(1)
		}
		if !supportsService {
			handler = controller.NewTrivialHandler(clientset)
			klog.V(2).Infof("CSI driver does not support Plugin Controller Service, using trivial handler")
		} else {
			// Find out if the driver supports attach/detach.
			supportsAttach, supportsReadOnly, err := csiConn.SupportsControllerPublish(ctx)
			if err != nil {
				klog.Error(err.Error())
				os.Exit(1)
			}
			if supportsAttach {
				pvLister := factory.Core().V1().PersistentVolumes().Lister()
				nodeLister := factory.Core().V1().Nodes().Lister()
				vaLister := factory.Storage().V1beta1().VolumeAttachments().Lister()
				csiFactory := csiinformers.NewSharedInformerFactory(csiClientset, *resync)
				nodeInfoLister := csiFactory.Csi().V1alpha1().CSINodeInfos().Lister()
				handler = controller.NewCSIHandler(clientset, csiClientset, attacher, csiConn, pvLister, nodeLister, nodeInfoLister, vaLister, timeout, supportsReadOnly)
				klog.V(2).Infof("CSI driver supports ControllerPublishUnpublish, using real CSI handler")
			} else {
				handler = controller.NewTrivialHandler(clientset)
				klog.V(2).Infof("CSI driver does not support ControllerPublishUnpublish, using trivial handler")
			}
		}
	}

	ctrl := controller.NewCSIAttachController(
		clientset,
		attacher,
		handler,
		factory.Storage().V1beta1().VolumeAttachments(),
		factory.Core().V1().PersistentVolumes(),
	)

	run := func(ctx context.Context) {
		stopCh := ctx.Done()
		factory.Start(stopCh)
		if csiFactory != nil {
			csiFactory.Start(stopCh)
		}
		ctrl.Run(threads, stopCh)
	}

	if !*enableLeaderElection {
		run(context.TODO())
	} else {
		// Leader election was requested.
		if leaderElectionNamespace == nil || *leaderElectionNamespace == "" {
			klog.Error("-leader-election-namespace must not be empty")
			os.Exit(1)
		}
		if leaderElectionIdentity == nil || *leaderElectionIdentity == "" {
			klog.Error("-leader-election-identity must not be empty")
			os.Exit(1)
		}
		// Name of config map with leader election lock
		lockName := "external-attacher-leader-" + attacher
		runAsLeader(clientset, *leaderElectionNamespace, *leaderElectionIdentity, lockName, run)
	}
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
