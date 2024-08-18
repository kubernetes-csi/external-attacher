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
	"github.com/kubernetes-csi/external-attacher/pkg/controller"
	"google.golang.org/grpc"
)

const (

	// Default timeout of short CSI calls like GetPluginInfo
	csiTimeout = time.Second
)

// common command flags
var (
	kubeconfig                  string
	csiAddress                  string
	showVersion                 bool
	httpEndpoint                string
	enableLeaderElection        bool
	leaderElectionNamespace     string
	leaderElectionLeaseDuration time.Duration
	leaderElectionRenewDeadline time.Duration
	leaderElectionRetryPeriod   time.Duration
)

// attacher command line flags
var (
	resync             time.Duration
	timeout            time.Duration
	workerThreads      uint64
	maxEntries         int
	retryIntervalStart time.Duration
	retryIntervalMax   time.Duration

	defaultFSType string
	reconcileSync time.Duration

	metricsAddress string
	metricsPath    string

	kubeAPIQPS   float64
	kubeAPIBurst int

	maxGRPCLogLength int
)

func initCommonFlags() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file. Required only when running out of cluster.")
	flag.StringVar(&csiAddress, "csi-address", "/run/csi/socket", "Address of the CSI driver socket.")

	// I think we should remove deprecated flag in AIO project, after all we must change the non-common flags name such as worker-thread to attacher-worker-thread.
	flag.StringVar(&metricsAddress, "metrics-address", "", "(deprecated) The TCP network address where the prometheus metrics endpoint will listen (example: `:8080`). The default is empty string, which means metrics endpoint is disabled. Only one of `--metrics-address` and `--http-endpoint` can be set.")
	flag.StringVar(&httpEndpoint, "http-endpoint", "", "The TCP network address where the HTTP server for diagnostics, including metrics and leader election health check, will listen (example: `:8080`). The default is empty string, which means the server is disabled. Only one of `http-endpoint` and `metrics-address` can be set.")

	flag.BoolVar(&enableLeaderElection, "leader-election", false, "Enable leader election.")
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "", "Namespace where the leader election resource lives. Defaults to the pod namespace if not set.")
	flag.DurationVar(&leaderElectionLeaseDuration, "leader-election-lease-duration", 15*time.Second, "Duration, in seconds, that non-leader candidates will wait to force acquire leadership. Defaults to 15 seconds.")
	flag.DurationVar(&leaderElectionRenewDeadline, "leader-election-renew-deadline", 10*time.Second, "Duration, in seconds, that the acting leader will retry refreshing leadership before giving up. Defaults to 10 seconds.")
	flag.DurationVar(&leaderElectionRetryPeriod, "leader-election-retry-period", 2*time.Second, "Duration, in seconds, the LeaderElector clients should wait between tries of actions. Defaults to 2 seconds.")
}

func initAttacherFlags() {
	flag.DurationVar(&resync, "resync", 10*time.Minute, "Resync interval of the controller.")
	flag.DurationVar(&timeout, "timeout", 15*time.Second, "Timeout for waiting for attaching or detaching the volume.")
	flag.Uint64Var(&workerThreads, "worker-threads", 10, "Number of attacher worker threads")
	flag.IntVar(&maxEntries, "max-entries", 0, "Max entries per each page in volume lister call, 0 means no limit.")

	flag.DurationVar(&retryIntervalStart, "retry-interval-start", time.Second, "Initial retry interval of failed create volume or deletion. It doubles with each failure, up to retry-interval-max.")
	flag.DurationVar(&retryIntervalMax, "retry-interval-max", 5*time.Minute, "Maximum retry interval of failed create volume or deletion.")

	flag.StringVar(&defaultFSType, "default-fstype", "", "The default filesystem type of the volume to publish. Defaults to empty string")
	flag.DurationVar(&reconcileSync, "reconcile-sync", 1*time.Minute, "Resync interval of the VolumeAttachment reconciler.")

	flag.Float64Var(&kubeAPIQPS, "kube-api-qps", 5, "QPS to use while communicating with the kubernetes apiserver. Defaults to 5.0.")
	flag.IntVar(&kubeAPIBurst, "kube-api-burst", 10, "Burst to use while communicating with the kubernetes apiserver. Defaults to 10.")

	flag.IntVar(&maxGRPCLogLength, "max-grpc-log-length", -1, "The maximum amount of characters logged for every grpc responses. Defaults to no limit")
}

func printVersion(logger klog.Logger) {
	logger.Info("Version", "version", version)
	if showVersion {
		fmt.Println(os.Args[0], version)
		return
	}
}

func getMetricsAddr(logger klog.Logger) (addr string) {
	if metricsAddress != "" && httpEndpoint != "" {
		logger.Error(nil, "Only one of `--metrics-address` and `--http-endpoint` can be set")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	addr = metricsAddress
	if addr == "" {
		addr = httpEndpoint
	}
	return
}

func buildKubeConfig(logger klog.Logger) *rest.Config {
	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	var config *rest.Config
	var err error
	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		logger.Error(err, "Failed to build a Kubernetes config")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	config.QPS = (float32)(kubeAPIQPS)
	config.Burst = kubeAPIBurst
	return config
}

func findDriverNameAndCSIConn(ctx context.Context, metricsManager metrics.CSIMetricsManager) (string, *grpc.ClientConn) {
	logger := ctx.Value("logger").(klog.Logger)
	// Connect to CSI.
	connection.SetMaxGRPCLogLength(maxGRPCLogLength)
	csiConn, err := connection.Connect(ctx, csiAddress, metricsManager, connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
	if err != nil {
		logger.Error(err, "Failed to connect to the CSI driver", "csiAddress", csiAddress)
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	err = rpc.ProbeForever(ctx, csiConn, timeout)
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
		migratedCsiClient, err := connection.Connect(ctx, csiAddress, metricsManager, connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
		if err != nil {
			logger.Error(err, "Failed to connect to the CSI driver", "csiAddress", csiAddress, "migrated", true)
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		csiConn.Close()
		csiConn = migratedCsiClient

		err = rpc.ProbeForever(ctx, csiConn, timeout)
		if err != nil {
			logger.Error(err, "Failed to probe the CSI driver", "migrated", true)
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}
	return csiAttacher, csiConn
}

func getSupportService(ctx context.Context, csiConn *grpc.ClientConn) (supportService, supportsAttach, supportsReadOnly, supportsListVolumesPublishedNodes, supportsSingleNodeMultiWriter bool) {

	logger := ctx.Value("logger").(klog.Logger)
	cancelationCtx, cancel := context.WithTimeout(ctx, csiTimeout)
	cancelationCtx = klog.NewContext(cancelationCtx, logger)
	defer cancel()
	supportService, err := supportsPluginControllerService(cancelationCtx, csiConn)
	if err != nil {
		logger.Error(err, "Failed to check if the CSI Driver supports the CONTROLLER_SERVICE")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	if supportService {
		supportsAttach, supportsReadOnly, supportsListVolumesPublishedNodes, supportsSingleNodeMultiWriter, err = supportsControllerCapabilities(cancelationCtx, csiConn)
		if err != nil {
			logger.Error(err, "Failed to controller capability check")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}
	return
}

func NewAttacherCSIHandler(
	csiAttacher string,
	supportsService,
	supportsAttach,
	supportsReadOnly,
	supportsSingleNodeMultiWriter bool,
	clientset *kubernetes.Clientset,
	csiConn *grpc.ClientConn,
	factory informers.SharedInformerFactory,
	logger klog.Logger) controller.Handler {
	var handler controller.Handler
	if !supportsService {
		handler = controller.NewTrivialHandler(clientset)
		logger.V(2).Info("CSI driver does not support Plugin Controller Service, using trivial handler")
	} else {
		if supportsAttach {
			pvLister := factory.Core().V1().PersistentVolumes().Lister()
			vaLister := factory.Storage().V1().VolumeAttachments().Lister()
			csiNodeLister := factory.Storage().V1().CSINodes().Lister()
			volAttacher := attacher.NewAttacher(csiConn)
			CSIVolumeLister := attacher.NewVolumeLister(csiConn, maxEntries)
			handler = controller.NewCSIHandler(
				clientset,
				csiAttacher,
				volAttacher,
				CSIVolumeLister,
				pvLister,
				csiNodeLister,
				vaLister,
				&timeout,
				supportsReadOnly,
				supportsSingleNodeMultiWriter,
				csitrans.New(),
				defaultFSType,
			)
			logger.V(2).Info("CSI driver supports ControllerPublishUnpublish, using real CSI handler")
		} else {
			handler = controller.NewTrivialHandler(clientset)
			logger.V(2).Info("CSI driver does not support ControllerPublishUnpublish, using trivial handler")
		}
	}
	return handler
}

func startController(lockName string, config *rest.Config, run func(ctx context.Context), mux *http.ServeMux, logger klog.Logger) {
	if !enableLeaderElection {
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

		le := leaderelection.NewLeaderElection(leClientset, lockName, run)
		if httpEndpoint != "" {
			le.PrepareHealthCheck(mux, leaderelection.DefaultHealthCheckTimeout)
		}

		if leaderElectionNamespace != "" {
			le.WithNamespace(leaderElectionNamespace)
		}

		le.WithLeaseDuration(leaderElectionLeaseDuration)
		le.WithRenewDeadline(leaderElectionRenewDeadline)
		le.WithRetryPeriod(leaderElectionRetryPeriod)

		if err := le.Run(); err != nil {
			logger.Error(err, "Failed to initialize leader election")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}
}

var (
	version = "unknown"
)

func main() {
	initCommonFlags()
	initAttacherFlags()
	fg := featuregate.NewFeatureGate()
	logsapi.AddFeatureGates(fg)
	c := logsapi.NewLoggingConfiguration()
	logsapi.AddGoFlags(c, flag.CommandLine)
	logs.InitLogs()
	flag.Parse()
	logger := klog.Background()
	if err := logsapi.ValidateAndApply(c, fg); err != nil {
		logger.Error(err, "LoggingConfiguration is invalid")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	printVersion(logger)
	addr := getMetricsAddr(logger)

	if workerThreads == 0 {
		logger.Error(nil, "Option -worker-threads must be greater than zero")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	config := buildKubeConfig(logger)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create a Clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	factory := informers.NewSharedInformerFactory(clientset, resync)
	metricsManager := metrics.NewCSIMetricsManager("" /* driverName */)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "logger", logger)

	csiAttacher, csiConn := findDriverNameAndCSIConn(ctx, metricsManager)

	// Prepare http endpoint for metrics + leader election healthz
	mux := http.NewServeMux()
	if addr != "" {
		metricsManager.RegisterToServer(mux, metricsPath)
		metricsManager.SetDriverName(csiAttacher)
		go func() {
			logger.Info("ServeMux listening", "address", addr, "metricsPath", metricsPath)
			err := http.ListenAndServe(addr, mux)
			if err != nil {
				logger.Error(err, "Failed to start HTTP server at specified address and metrics path", "address", addr, "metricsPath", metricsPath)
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}()
	}
	supportsService, supportsAttach, supportsReadOnly, supportsListVolumesPublishedNodes, supportsSingleNodeMultiWriter := getSupportService(ctx, csiConn)

	if supportsListVolumesPublishedNodes {
		logger.V(2).Info("CSI driver supports list volumes published nodes. Using capability to reconcile volume attachment objects with actual backend state")
	}
	handler := NewAttacherCSIHandler(csiAttacher, supportsService, supportsAttach, supportsReadOnly, supportsSingleNodeMultiWriter, clientset, csiConn, factory, logger)

	ctrl := controller.NewCSIAttachController(
		logger,
		clientset,
		csiAttacher,
		handler,
		factory.Storage().V1().VolumeAttachments(),
		factory.Core().V1().PersistentVolumes(),
		workqueue.NewItemExponentialFailureRateLimiter(retryIntervalStart, retryIntervalMax),
		workqueue.NewItemExponentialFailureRateLimiter(retryIntervalStart, retryIntervalMax),
		supportsListVolumesPublishedNodes,
		reconcileSync,
	)

	run := func(ctx context.Context) {
		stopCh := ctx.Done()
		factory.Start(stopCh)
		ctrl.Run(ctx, int(workerThreads))
	}
	// Name of config map with leader election lock
	lockName := "external-attacher-leader-" + csiAttacher
	startController(lockName, config, run, mux, logger)

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
