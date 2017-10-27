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
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/kubernetes-csi/external-attacher-csi/pkg/controller"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// Number of worker threads
	threads = 10
)

var (
	// TODO: add a better description of attacher name - is it CSI plugin name? Kubernetes plugin name?
	kubeconfig        = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Required only when running out of cluster.")
	resync            = flag.Duration("resync", 10*time.Second, "Resync interval of the controller.")
	connectionTimeout = flag.Duration("connection-timeout", 60, "Timeout for waiting for CSI driver socket (in seconds).")
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	factory := informers.NewSharedInformerFactory(clientset, *resync)

	// TODO: wait for CSI's socket and discover 'attacher' and whether the
	// driver supports ControllerPulishVolume using ControllerGetCapabilities
	attacher := "csi/example"
	handler := controller.NewTrivialHandler(clientset)

	// Start the provision controller which will dynamically provision NFS PVs
	ctrl := controller.NewCSIAttachController(
		clientset,
		attacher,
		handler,
		factory.Storage().V1().VolumeAttachments(),
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
