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
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/component-base/featuregate"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	_ "k8s.io/component-base/logs/json/register"
	csitrans "k8s.io/csi-translation-lib"
	"k8s.io/klog/v2"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"
	"github.com/kubernetes-csi/csi-lib-utils/metrics"
	"github.com/kubernetes-csi/csi-lib-utils/rpc"
	"github.com/kubernetes-csi/external-attacher/pkg/attacher"
	cf "github.com/kubernetes-csi/external-attacher/pkg/commandflags"
	"github.com/kubernetes-csi/external-attacher/pkg/controller"
	"google.golang.org/grpc"
)

const (

	// Default timeout of short CSI calls like GetPluginInfo
	csiTimeout = time.Second
)

var (
	version = "unknown"
)

func main() {
	fg := featuregate.NewFeatureGate()
	logsapi.AddFeatureGates(fg)
	cf.InitCommonFlags()
	cf.InitAttacherFlags()
	c := logsapi.NewLoggingConfiguration()
	logsapi.AddGoFlags(c, flag.CommandLine)
	logs.InitLogs()
	flag.Parse()
	logger := klog.Background()
	if err := logsapi.ValidateAndApply(c, fg); err != nil {
		logger.Error(err, "LoggingConfiguration is invalid")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	if cf.ShowVersion {
		fmt.Println(os.Args[0], version)
		return
	}
	logger.Info("Version", "version", version)

	if cf.MetricsAddress != "" && cf.HttpEndpoint != "" {
		logger.Error(nil, "Only one of `--metrics-address` and `--http-endpoint` can be set")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	addr := cf.MetricsAddress
	if addr == "" {
		addr = cf.HttpEndpoint
	}

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(cf.Kubeconfig)
	if err != nil {
		logger.Error(err, "Failed to build a Kubernetes config")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	config.QPS = (float32)(cf.KubeAPIQPS)
	config.Burst = cf.KubeAPIBurst
	config.ContentType = runtime.ContentTypeProtobuf

	if cf.WorkerThreads == 0 {
		logger.Error(nil, "Option -worker-threads must be greater than zero")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create a Clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	factory := informers.NewSharedInformerFactory(clientset, cf.Resync)
	var handler controller.Handler
	metricsManager := metrics.NewCSIMetricsManager("" /* driverName */)

	// Connect to CSI.
	connection.SetMaxGRPCLogLength(cf.MaxGRPCLogLength)
	ctx := context.Background()
	csiConn, err := connection.Connect(ctx, cf.CsiAddress, metricsManager, connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
	if err != nil {
		logger.Error(err, "Failed to connect to the CSI driver", "csiAddress", cf.CsiAddress)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	err = rpc.ProbeForever(ctx, csiConn, cf.Timeout)
	if err != nil {
		logger.Error(err, "Failed to probe the CSI driver")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	// Find driver name.
	cancelationCtx, cancel := context.WithTimeout(ctx, csiTimeout)
	cancelationCtx = klog.NewContext(cancelationCtx, logger)
	defer cancel()
	csiAttacher, err := rpc.GetDriverName(cancelationCtx, csiConn)
	if err != nil {
		logger.Error(err, "Failed to get the CSI driver name")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	logger = klog.LoggerWithValues(logger, "driver", csiAttacher)
	logger.V(2).Info("CSI driver name")

	translator := csitrans.New()
	if translator.IsMigratedCSIDriverByName(csiAttacher) {
		metricsManager = metrics.NewCSIMetricsManagerWithOptions(csiAttacher, metrics.WithMigration())
		migratedCsiClient, err := connection.Connect(ctx, cf.CsiAddress, metricsManager, connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
		if err != nil {
			logger.Error(err, "Failed to connect to the CSI driver", "csiAddress", cf.CsiAddress, "migrated", true)
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		csiConn.Close()
		csiConn = migratedCsiClient

		err = rpc.ProbeForever(ctx, csiConn, cf.Timeout)
		if err != nil {
			logger.Error(err, "Failed to probe the CSI driver", "migrated", true)
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}

	// Prepare http endpoint for metrics + leader election healthz
	mux := http.NewServeMux()
	if addr != "" {
		metricsManager.RegisterToServer(mux, cf.MetricsPath)
		metricsManager.SetDriverName(csiAttacher)
		go func() {
			logger.Info("ServeMux listening", "address", addr, "metricsPath", cf.MetricsPath)
			err := http.ListenAndServe(addr, mux)
			if err != nil {
				logger.Error(err, "Failed to start HTTP server at specified address and metrics path", "address", addr, "metricsPath", cf.MetricsPath)
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}()
	}

	cancelationCtx, cancel = context.WithTimeout(ctx, csiTimeout)
	cancelationCtx = klog.NewContext(cancelationCtx, logger)
	defer cancel()
	supportsService, err := supportsPluginControllerService(cancelationCtx, csiConn)
	if err != nil {
		logger.Error(err, "Failed to check if the CSI Driver supports the CONTROLLER_SERVICE")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	var (
		supportsAttach                    bool
		supportsReadOnly                  bool
		supportsListVolumesPublishedNodes bool
		supportsSingleNodeMultiWriter     bool
	)
	if !supportsService {
		handler = controller.NewTrivialHandler(clientset)
		logger.V(2).Info("CSI driver does not support Plugin Controller Service, using trivial handler")
	} else {
		supportsAttach, supportsReadOnly, supportsListVolumesPublishedNodes, supportsSingleNodeMultiWriter, err = supportsControllerCapabilities(cancelationCtx, csiConn)
		if err != nil {
			logger.Error(err, "Failed to controller capability check")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}

		if supportsAttach {
			pvLister := factory.Core().V1().PersistentVolumes().Lister()
			vaLister := factory.Storage().V1().VolumeAttachments().Lister()
			csiNodeLister := factory.Storage().V1().CSINodes().Lister()
			volAttacher := attacher.NewAttacher(csiConn)
			CSIVolumeLister := attacher.NewVolumeLister(csiConn, cf.MaxEntries)
			handler = controller.NewCSIHandler(
				clientset,
				csiAttacher,
				volAttacher,
				CSIVolumeLister,
				pvLister,
				csiNodeLister,
				vaLister,
				&cf.Timeout,
				supportsReadOnly,
				supportsSingleNodeMultiWriter,
				csitrans.New(),
				cf.DefaultFSType,
			)
			logger.V(2).Info("CSI driver supports ControllerPublishUnpublish, using real CSI handler")
		} else {
			handler = controller.NewTrivialHandler(clientset)
			logger.V(2).Info("CSI driver does not support ControllerPublishUnpublish, using trivial handler")
		}
	}

	if supportsListVolumesPublishedNodes {
		logger.V(2).Info("CSI driver supports list volumes published nodes. Using capability to reconcile volume attachment objects with actual backend state")
	}

	ctrl := controller.NewCSIAttachController(
		logger,
		clientset,
		csiAttacher,
		handler,
		factory.Storage().V1().VolumeAttachments(),
		factory.Core().V1().PersistentVolumes(),
		workqueue.NewItemExponentialFailureRateLimiter(cf.RetryIntervalStart, cf.RetryIntervalMax),
		workqueue.NewItemExponentialFailureRateLimiter(cf.RetryIntervalStart, cf.RetryIntervalMax),
		supportsListVolumesPublishedNodes,
		cf.ReconcileSync,
	)

	run := func(ctx context.Context) {
		stopCh := ctx.Done()
		factory.Start(stopCh)
		ctrl.Run(ctx, int(cf.WorkerThreads))
	}

	if !cf.EnableLeaderElection {
		run(klog.NewContext(context.Background(), logger))
	} else {
		// Create a new clientset for leader election. When the attacher
		// gets busy and its client gets throttled, the leader election
		// can proceed without issues.
		leClientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			logger.Error(err, "Failed to create leaderelection client")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}

		// Name of config map with leader election lock
		lockName := "external-attacher-leader-" + csiAttacher
		le := leaderelection.NewLeaderElection(leClientset, lockName, run)
		if cf.HttpEndpoint != "" {
			le.PrepareHealthCheck(mux, leaderelection.DefaultHealthCheckTimeout)
		}

		if cf.LeaderElectionNamespace != "" {
			le.WithNamespace(cf.LeaderElectionNamespace)
		}

		le.WithLeaseDuration(cf.LeaderElectionLeaseDuration)
		le.WithRenewDeadline(cf.LeaderElectionRenewDeadline)
		le.WithRetryPeriod(cf.LeaderElectionRetryPeriod)

		if err := le.Run(); err != nil {
			logger.Error(err, "Failed to initialize leader election")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func supportsControllerCapabilities(ctx context.Context, csiConn *grpc.ClientConn) (bool, bool, bool, bool, error) {
	caps, err := rpc.GetControllerCapabilities(ctx, csiConn)
	if err != nil {
		return false, false, false, false, err
	}

	supportsControllerPublish := caps[csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME]
	supportsPublishReadOnly := caps[csi.ControllerServiceCapability_RPC_PUBLISH_READONLY]
	supportsListVolumesPublishedNodes := caps[csi.ControllerServiceCapability_RPC_LIST_VOLUMES] && caps[csi.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES]
	supportsSingleNodeMultiWriter := caps[csi.ControllerServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER]
	return supportsControllerPublish, supportsPublishReadOnly, supportsListVolumesPublishedNodes, supportsSingleNodeMultiWriter, nil
}

func supportsPluginControllerService(ctx context.Context, csiConn *grpc.ClientConn) (bool, error) {
	caps, err := rpc.GetPluginCapabilities(ctx, csiConn)
	if err != nil {
		return false, err
	}

	return caps[csi.PluginCapability_Service_CONTROLLER_SERVICE], nil
}
