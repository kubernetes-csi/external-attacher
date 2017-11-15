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
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

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

		// Find out if the driver supports attach/detach.
		supportsAttach, err := csiConn.SupportsControllerPublish(ctx)
		if err != nil {
			glog.Error(err.Error())
			os.Exit(1)
		}
		if !supportsAttach {
			handler = controller.NewTrivialHandler(clientset)
			glog.V(2).Infof("CSI driver does not support ControllerPublishUnpublish, using trivial handler")
		} else {
			pvLister := factory.Core().V1().PersistentVolumes().Lister()
			nodeLister := factory.Core().V1().Nodes().Lister()
			vaLister := factory.Storage().V1alpha1().VolumeAttachments().Lister()
			handler = controller.NewCSIHandler(clientset, attacher, csiConn, pvLister, nodeLister, vaLister)
			glog.V(2).Infof("CSI driver supports ControllerPublishUnpublish, using real CSI handler")
		}
	}

	ctrl := controller.NewCSIAttachController(
		clientset,
		attacher,
		handler,
		factory.Storage().V1alpha1().VolumeAttachments(),
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
