package commandflags

import (
	"flag"
	"time"
)

// attacher command line flags
var (
	Resync             time.Duration
	Timeout            time.Duration
	WorkerThreads      uint64
	MaxEntries         int
	RetryIntervalStart time.Duration
	RetryIntervalMax   time.Duration

	DefaultFSType string
	ReconcileSync time.Duration

	MetricsPath string

	KubeAPIQPS   float64
	KubeAPIBurst int

	MaxGRPCLogLength int
)

func InitAttacherFlags() {
	initResync()
	initTimeout()
	initWorkerThreads()
	initMaxEntries()

	initRetryIntervalStart()
	initRetryIntervalMax()

	initDefaultFSType()

	initKubeAPIQPS()
	initKubeAPIBurst()

	initMaxGPRCLogLength()
}

func initResync() {
	flag.DurationVar(&Resync, "attacher-resync", -1*time.Minute, "Resync interval of the controller.")
	if Resync == -1*time.Minute {
		flag.DurationVar(&Resync, "resync", 10*time.Minute, "Resync interval of the controller.")
	}
}

func initTimeout() {
	flag.DurationVar(&Timeout, "attacher-timeout", -1*time.Second, "Timeout for waiting for attaching or detaching the volume.")
	if Timeout == -1*time.Minute {
		flag.DurationVar(&Timeout, "timeout", 15*time.Second, "Timeout for waiting for attaching or detaching the volume.")
	}
}

func initWorkerThreads() {
	flag.Uint64Var(&WorkerThreads, "attacher-worker-threads", 0, "Number of attacher worker threads")
	if WorkerThreads == 0 {
		flag.Uint64Var(&WorkerThreads, "worker-threads", 10, "Number of attacher worker threads")
	}
}

func initMaxEntries() {
	flag.IntVar(&MaxEntries, "attacher-max-entries", -1, "Max entries per each page in volume lister call, 0 means no limit.")
	if MaxEntries == -1 {
		flag.IntVar(&MaxEntries, "max-entries", 0, "Max entries per each page in volume lister call, 0 means no limit.")
	}
}

func initRetryIntervalStart() {
	flag.DurationVar(&RetryIntervalStart, "attacher-retry-interval-start", -1*time.Second, "Initial retry interval of failed create volume or deletion. It doubles with each failure, up to retry-interval-max.")
	if RetryIntervalStart == -1*time.Second {
		flag.DurationVar(&RetryIntervalStart, "retry-interval-start", time.Second, "Initial retry interval of failed create volume or deletion. It doubles with each failure, up to retry-interval-max.")
	}
}

func initRetryIntervalMax() {
	flag.DurationVar(&RetryIntervalMax, "attacher-retry-interval-max", -1*time.Minute, "Maximum retry interval of failed create volume or deletion.")
	if RetryIntervalMax == -1*time.Minute {
		flag.DurationVar(&RetryIntervalMax, "retry-interval-max", 5*time.Minute, "Maximum retry interval of failed create volume or deletion.")
	}
}

func initDefaultFSType() {
	flag.StringVar(&DefaultFSType, "attacher-default-fstype", "", "The default filesystem type of the volume to publish. Defaults to empty string")
	if DefaultFSType == "" {
		flag.StringVar(&DefaultFSType, "default-fstype", "", "The default filesystem type of the volume to publish. Defaults to empty string")
	}
}

func initReconcileSync() {
	flag.DurationVar(&ReconcileSync, "attacher-reconcile-sync", -1*time.Minute, "Resync interval of the VolumeAttachment reconciler.")
	if ReconcileSync == -1*time.Minute {
		flag.DurationVar(&ReconcileSync, "reconcile-sync", 1*time.Minute, "Resync interval of the VolumeAttachment reconciler.")
	}
}

func initKubeAPIQPS() {
	flag.Float64Var(&KubeAPIQPS, "attacher-kube-api-qps", -1, "QPS to use while communicating with the kubernetes apiserver. Defaults to 5.0.")
	if KubeAPIQPS == -1 {
		flag.Float64Var(&KubeAPIQPS, "kube-api-qps", 5, "QPS to use while communicating with the kubernetes apiserver. Defaults to 5.0.")
	}
}

func initKubeAPIBurst() {
	flag.IntVar(&KubeAPIBurst, "attacher-kube-api-burst", -1, "Burst to use while communicating with the kubernetes apiserver. Defaults to 10.")
	if KubeAPIBurst == -1 {
		flag.IntVar(&KubeAPIBurst, "kube-api-burst", 10, "Burst to use while communicating with the kubernetesapiserver. Defaults to 10.")
	}
}

func initMaxGPRCLogLength() {
	flag.IntVar(&MaxGRPCLogLength, "attacher-max-gprc-log-length", -1, "The maximum amount of characters logged for every grpc responses. Defaults to no limit")
	if MaxGRPCLogLength == -1 {
		flag.IntVar(&MaxGRPCLogLength, "max-grpc-log-length", -1, "The maximum amount of characters logged for every grpc responses. Defaults to no limit")
	}
}
