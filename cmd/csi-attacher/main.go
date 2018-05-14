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
	"os/signal"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

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
	resync            = flag.Duration("resync", 10*time.Second, "Resync interval of the controller.")
	connectionTimeout = flag.Duration("connection-timeout", 1*time.Minute, "Timeout for waiting for CSI driver socket.")
	csiAddress        = flag.String("csi-address", "/run/csi/socket", "Address of the CSI driver socket.")
	dummy             = flag.Bool("dummy", false, "Run in dummy mode, i.e. not connecting to CSI driver and marking everything as attached. Expected CSI driver name is \"csi/dummy\".")
	showVersion       = flag.Bool("version", false, "Show version.")

	enableLeaderElection    = flag.Bool("leader-election", false, "Enable leader election.")
	leaderElectionNamespace = flag.String("leader-election-namespace", "", "Namespace where this attacher runs.")
	leaderElectionIdentity  = flag.String("leader-election-identity", "", "Unique idenity of this attcher. Typically name of the pod where the attacher runs.")
)

var (
	version = "unknown"
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	if *showVersion {
		fmt.Println(os.Args[0], version)
		return
	}
	glog.Infof("Version: %s", version)

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		glog.Error(err.Error())
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Error(err.Error())
		os.Exit(1)
	}

	factory := informers.NewSharedInformerFactory(clientset, *resync)

	var handler controller.Handler

	var attacher string
	if *dummy {
		// Do not connect to any CSI, mark everything as attached.
		handler = controller.NewTrivialHandler(clientset)
		attacher = dummyAttacherName
	} else {
		// Connect to CSI.
		csiConn, err := connection.New(*csiAddress, *connectionTimeout)
		if err != nil {
			glog.Error(err.Error())
			os.Exit(1)
		}

		// Find driver name.
		ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
		defer cancel()
		attacher, err = csiConn.GetDriverName(ctx)
		if err != nil {
			glog.Error(err.Error())
			os.Exit(1)
		}
		glog.V(2).Infof("CSI driver name: %q", attacher)

		// Check it's ready
		if err = waitForDriverReady(csiConn, *connectionTimeout); err != nil {
			glog.Error(err.Error())
			os.Exit(1)
		}

		supportsService, err := csiConn.SupportsPluginControllerService(ctx)
		if err != nil {
			glog.Error(err.Error())
			os.Exit(1)
		}
		if !supportsService {
			handler = controller.NewTrivialHandler(clientset)
			glog.V(2).Infof("CSI driver does not support Plugin Controller Service, using trivial handler")
		} else {
			// Find out if the driver supports attach/detach.
			supportsAttach, err := csiConn.SupportsControllerPublish(ctx)
			if err != nil {
				glog.Error(err.Error())
				os.Exit(1)
			}
			if supportsAttach {
				pvLister := factory.Core().V1().PersistentVolumes().Lister()
				nodeLister := factory.Core().V1().Nodes().Lister()
				vaLister := factory.Storage().V1beta1().VolumeAttachments().Lister()
				handler = controller.NewCSIHandler(clientset, attacher, csiConn, pvLister, nodeLister, vaLister)
				glog.V(2).Infof("CSI driver supports ControllerPublishUnpublish, using real CSI handler")
			} else {
				handler = controller.NewTrivialHandler(clientset)
				glog.V(2).Infof("CSI driver does not support ControllerPublishUnpublish, using trivial handler")
			}
		}
	}

	if *enableLeaderElection {
		// Leader election was requested.
		if leaderElectionNamespace == nil || *leaderElectionNamespace == "" {
			glog.Error("-leader-election-namespace must not be empty")
			os.Exit(1)
		}
		if leaderElectionIdentity == nil || *leaderElectionIdentity == "" {
			glog.Error("-leader-election-identity must not be empty")
			os.Exit(1)
		}
		// Name of config map with leader election lock
		lockName := "external-attacher-leader-" + attacher
		waitForLeader(clientset, *leaderElectionNamespace, *leaderElectionIdentity, lockName)
	}

	ctrl := controller.NewCSIAttachController(
		clientset,
		attacher,
		handler,
		factory.Storage().V1beta1().VolumeAttachments(),
		factory.Core().V1().PersistentVolumes(),
	)

	// run...
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	go ctrl.Run(threads, stopCh)

	// ...until SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	close(stopCh)
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func waitForDriverReady(csiConn connection.CSIConnection, timeout time.Duration) error {
	now := time.Now()
	finish := now.Add(timeout)
	var err error
	for {
		ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
		defer cancel()
		err = csiConn.Probe(ctx)
		if err == nil {
			glog.V(2).Infof("Probe succeeded")
			return nil
		}
		glog.V(2).Infof("Probe failed with %s", err)

		now := time.Now()
		if now.After(finish) {
			return fmt.Errorf("Failed to probe the controller: %s", err)
		}
		time.Sleep(time.Second)
	}
}
