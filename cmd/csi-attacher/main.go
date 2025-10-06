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
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/server"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/featuregate"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	_ "k8s.io/component-base/logs/json/register"
	"k8s.io/component-base/metrics/legacyregistry"
	_ "k8s.io/component-base/metrics/prometheus/clientgo/leaderelection" // register leader election in the default legacy registry
	_ "k8s.io/component-base/metrics/prometheus/workqueue"               // register work queues in the default legacy registry
	csitrans "k8s.io/csi-translation-lib"
	"k8s.io/klog/v2"

	"github.com/container-storage-interface/spec/lib/go/csi"
	libconfig "github.com/kubernetes-csi/csi-lib-utils/config"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"
	"github.com/kubernetes-csi/csi-lib-utils/metrics"
	"github.com/kubernetes-csi/csi-lib-utils/rpc"
	"github.com/kubernetes-csi/csi-lib-utils/standardflags"
	"github.com/kubernetes-csi/external-attacher/v4/pkg/attacher"
	"github.com/kubernetes-csi/external-attacher/v4/pkg/controller"
	"github.com/kubernetes-csi/external-attacher/v4/pkg/features"
	"google.golang.org/grpc"
)

const (

	// Default timeout of short CSI calls like GetPluginInfo
	csiTimeout = time.Second
)

// Command line flags
var (
	resync             = flag.Duration("resync", 10*time.Minute, "Resync interval of the controller.")
	timeout            = flag.Duration("timeout", 15*time.Second, "Timeout for waiting for attaching or detaching the volume.")
	retryIntervalStart = flag.Duration("retry-interval-start", time.Second, "Initial retry interval of failed provisioning or deletion. It doubles with each failure, up to retry-interval-max.")
	retryIntervalMax   = flag.Duration("retry-interval-max", 5*time.Minute, "Maximum retry interval of failed provisioning or deletion.")
	workerThreads      = flag.Uint("worker-threads", 10, "Number of attacher worker threads")
	maxEntries         = flag.Int("max-entries", 0, "Max entries per each page in volume lister call, 0 means no limit.")

	defaultFSType = flag.String("default-fstype", "", "The default filesystem type of the volume to publish. Defaults to empty string")

	reconcileSync = flag.Duration("reconcile-sync", 1*time.Minute, "Resync interval of the VolumeAttachment reconciler.")

	maxGRPCLogLength = flag.Int("max-grpc-log-length", -1, "The maximum amount of characters logged for every grpc responses. Defaults to no limit")

	featureGates map[string]bool
)

var (
	version = "unknown"
)

func main() {
	flag.Var(utilflag.NewMapStringBool(&featureGates), "feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features. "+
		"Options are:\n"+strings.Join(utilfeature.DefaultFeatureGate.KnownFeatures(), "\n"))

	fg := featuregate.NewFeatureGate()
	logsapi.AddFeatureGates(fg)
	c := logsapi.NewLoggingConfiguration()
	logsapi.AddGoFlags(c, flag.CommandLine)
	logs.InitLogs()
	standardflags.RegisterCommonFlags(flag.CommandLine)
	standardflags.AddAutomaxprocs(klog.Infof)
	flag.Parse()
	logger := klog.Background()
	if err := logsapi.ValidateAndApply(c, fg); err != nil {
		logger.Error(err, "LoggingConfiguration is invalid")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	if err := utilfeature.DefaultMutableFeatureGate.SetFromMap(featureGates); err != nil {
		logger.Error(err, "failed to store flag gates", "featureGates", featureGates)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	if standardflags.Configuration.ShowVersion {
		fmt.Println(os.Args[0], version)
		return
	}
	logger.Info("Version", "version", version)

	if standardflags.Configuration.MetricsAddress != "" && standardflags.Configuration.HttpEndpoint != "" {
		logger.Error(nil, "Only one of `--metrics-address` and `--http-endpoint` can be set")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	addr := standardflags.Configuration.MetricsAddress
	if addr == "" {
		addr = standardflags.Configuration.HttpEndpoint
	}

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := libconfig.BuildConfig(standardflags.Configuration.KubeConfig, standardflags.Configuration)
	if err != nil {
		logger.Error(err, "Failed to build a Kubernetes config")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	config.ContentType = runtime.ContentTypeProtobuf

	if *workerThreads == 0 {
		logger.Error(nil, "Option -worker-threads must be greater than zero")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create a Clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	factory := informers.NewSharedInformerFactory(clientset, *resync)
	var handler controller.Handler
	metricsManager := metrics.NewCSIMetricsManager("" /* driverName */)

	// Connect to CSI.
	connection.SetMaxGRPCLogLength(*maxGRPCLogLength)
	ctx := context.Background()
	csiConn, err := connection.Connect(ctx, standardflags.Configuration.CSIAddress, metricsManager, connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
	if err != nil {
		logger.Error(err, "Failed to connect to the CSI driver", "csiAddress", standardflags.Configuration.CSIAddress)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	err = rpc.ProbeForever(ctx, csiConn, *timeout)
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
		migratedCsiClient, err := connection.Connect(ctx, standardflags.Configuration.CSIAddress, metricsManager, connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
		if err != nil {
			logger.Error(err, "Failed to connect to the CSI driver", "csiAddress", standardflags.Configuration.CSIAddress, "migrated", true)
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		csiConn.Close()
		csiConn = migratedCsiClient

		err = rpc.ProbeForever(ctx, csiConn, *timeout)
		if err != nil {
			logger.Error(err, "Failed to probe the CSI driver", "migrated", true)
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}

	// Add default legacy registry so that metrics manager serves Go runtime and process metrics.
	// Also registers the `k8s.io/component-base/` work queue and leader election metrics we anonymously import.
	metricsManager.WithAdditionalRegistry(legacyregistry.DefaultGatherer)

	// Prepare http endpoint for metrics + leader election healthz
	mux := http.NewServeMux()
	if addr != "" {
		metricsManager.RegisterToServer(mux, standardflags.Configuration.MetricsPath)
		metricsManager.SetDriverName(csiAttacher)
		go func() {
			logger.Info("ServeMux listening", "address", addr, "metricsPath", standardflags.Configuration.MetricsPath)
			err := http.ListenAndServe(addr, mux)
			if err != nil {
				logger.Error(err, "Failed to start HTTP server at specified address and metrics path", "address", addr, "metricsPath", standardflags.Configuration.MetricsPath)
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
			CSIVolumeLister := attacher.NewVolumeLister(csiConn, *maxEntries)
			handler = controller.NewCSIHandler(
				clientset,
				csiAttacher,
				volAttacher,
				CSIVolumeLister,
				pvLister,
				csiNodeLister,
				vaLister,
				timeout,
				supportsReadOnly,
				supportsSingleNodeMultiWriter,
				csitrans.New(),
				*defaultFSType,
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
		workqueue.NewTypedItemExponentialFailureRateLimiter[string](*retryIntervalStart, *retryIntervalMax),
		workqueue.NewTypedItemExponentialFailureRateLimiter[string](*retryIntervalStart, *retryIntervalMax),
		supportsListVolumesPublishedNodes,
		*reconcileSync,
	)
	// handle SIGTERM and SIGINT by cancelling the context.
	var (
		terminate       func()          // called when all controllers are finished
		controllerCtx   context.Context // shuts down all controllers on a signal
		shutdownHandler <-chan struct{} // called when the signal is received
	)

	if utilfeature.DefaultFeatureGate.Enabled(features.ReleaseLeaderElectionOnExit) {
		ctx, terminate = context.WithCancel(ctx) // shuts down the whole process, incl. leader election
		var cancelControllerCtx context.CancelFunc
		controllerCtx, cancelControllerCtx = context.WithCancel(ctx)
		shutdownHandler = server.SetupSignalHandler()

		defer terminate()

		go func() {
			defer cancelControllerCtx()
			<-shutdownHandler
			logger.Info("Received SIGTERM or SIGINT signal, shutting down controller.")
		}()
	}

	run := func(ctx context.Context) {
		if utilfeature.DefaultFeatureGate.Enabled(features.ReleaseLeaderElectionOnExit) {
			var wg sync.WaitGroup
			factory.Start(shutdownHandler)
			ctrl.Run(controllerCtx, int(*workerThreads), &wg)
			wg.Wait()
			terminate()
		} else {
			stopCh := ctx.Done()
			factory.Start(stopCh)
			ctrl.Run(ctx, int(*workerThreads), nil)
		}
	}

	leaderelection.RunWithLeaderElection(
		ctx,
		config,
		standardflags.Configuration,
		run,
		"external-attacher-leader-"+csiAttacher,
		mux,
		utilfeature.DefaultFeatureGate.Enabled(features.ReleaseLeaderElectionOnExit),
	)
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
