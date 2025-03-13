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

func init() {
	flag.DurationVar(&Resync, "resync", 10*time.Minute, "Resync interval of the controller.")
	flag.DurationVar(&Timeout, "timeout", 15*time.Second, "Timeout for waiting for attaching or detaching the volume.")
	flag.Uint64Var(&WorkerThreads, "worker-threads", 10, "Number of attacher worker threads")
	flag.IntVar(&MaxEntries, "max-entries", 0, "Max entries per each page in volume lister call, 0 means no limit.")
	flag.DurationVar(&RetryIntervalStart, "retry-interval-start", time.Second, "Initial retry interval of failed create volume or deletion. It doubles with each failure, up to retry-interval-max.")
	flag.DurationVar(&RetryIntervalMax, "retry-interval-max", 5*time.Minute, "Maximum retry interval of failed create volume or deletion.")
	flag.StringVar(&DefaultFSType, "default-fstype", "", "The default filesystem type of the volume to publish. Defaults to empty string")
	flag.DurationVar(&ReconcileSync, "reconcile-sync", 1*time.Minute, "Resync interval of the VolumeAttachment reconciler.")
	flag.Float64Var(&KubeAPIQPS, "kube-api-qps", 5, "QPS to use while communicating with the kubernetes apiserver. Defaults to 5.0.")
	flag.IntVar(&KubeAPIBurst, "kube-api-burst", 10, "Burst to use while communicating with the kubernetesapiserver. Defaults to 10.")
	flag.IntVar(&MaxGRPCLogLength, "max-grpc-log-length", -1, "The maximum amount of characters logged for every grpc responses. Defaults to no limit")
	flag.StringVar(&MetricsPath, "metrics-path", "/metrics", "The TCP network address where the prometheus metrics endpoint will listen (example: `:8080`). Defaults to `/metrics`.")
}

type AttacherCommandFlags struct {
	resync             time.Duration
	timeout            time.Duration
	workerThreads      uint64
	maxEntries         int
	retryIntervalStart time.Duration
	retryIntervalMax   time.Duration
	maxGRPCLogLength   int
	kubeAPIQPS         float64
	kubeAPIBurst       int
	metricsPath        string
	defaultFSType      string
	reconcileSync      time.Duration
}

type SidecarControllerFlags interface {
	MergeFlags()
}

func NewAttacherCommandFlags() *AttacherCommandFlags {
	acf := AttacherCommandFlags{}
	flag.DurationVar(&acf.resync, "attacher-resync", -1*time.Minute, "Resync interval of the controller.")
	flag.DurationVar(&acf.timeout, "attacher-timeout", -1*time.Second, "Timeout for waiting for attaching or detaching the volume.")
	flag.Uint64Var(&acf.workerThreads, "attacher-worker-threads", 0, "Number of attacher worker threads")
	flag.IntVar(&acf.maxEntries, "attacher-max-entries", -1, "Max entries per each page in volume lister call, 0 means no limit.")
	flag.DurationVar(&acf.retryIntervalStart, "attacher-retry-interval-start", -1*time.Second, "Initial retry interval of failed create volume or deletion. It doubles with each failure, up to retry-interval-max.")
	flag.DurationVar(&acf.retryIntervalMax, "attacher-retry-interval-max", -1*time.Minute, "Maximum retry interval of failed create volume or deletion.")
	flag.StringVar(&acf.defaultFSType, "attacher-default-fstype", "", "The default filesystem type of the volume to publish. Defaults to empty string")
	flag.DurationVar(&acf.reconcileSync, "attacher-reconcile-sync", -1*time.Minute, "Resync interval of the VolumeAttachment reconciler.")
	flag.Float64Var(&acf.kubeAPIQPS, "attacher-kube-api-qps", -1, "QPS to use while communicating with the kubernetes apiserver. Defaults to 5.0.")
	flag.IntVar(&acf.kubeAPIBurst, "attacher-kube-api-burst", -1, "Burst to use while communicating with the kubernetes apiserver. Defaults to 10.")
	flag.IntVar(&acf.maxGRPCLogLength, "attacher-max-gprc-log-length", -1, "The maximum amount of characters logged for every grpc responses. Defaults to no limit")
	flag.StringVar(&acf.metricsPath, "attacher-metrics-path", "", "The TCP network address where the prometheus metrics endpoint will listen (example: `:8080`). Defaults to `/metrics`.")
	return &acf
}

func (acf *AttacherCommandFlags) MergeFlags() {
	acf.mergeResync()
	acf.mergeTimeout()
	acf.mergeWorkerThreads()
	acf.mergeMaxEntries()
	acf.mergeRetryIntervalStart()
	acf.mergeRetryIntervalMax()
	acf.mergeDefaultFSType()
	acf.mergeReconcileSync()
	acf.mergeKubeAPIQPS()
	acf.mergeKubeAPIBurst()
	acf.mergeMaxGPRCLogLength()
	acf.mergeMetricPath()

}

func (acf *AttacherCommandFlags) mergeResync() {
	if acf.resync != -1*time.Minute {
		Resync = acf.resync
	}
}

func (acf *AttacherCommandFlags) mergeTimeout() {
	if acf.timeout != -1*time.Second {
		Timeout = acf.timeout
	}
}

func (acf *AttacherCommandFlags) mergeWorkerThreads() {
	if acf.workerThreads != 0 {
		WorkerThreads = acf.workerThreads
	}
}

func (acf *AttacherCommandFlags) mergeMaxEntries() {
	if acf.maxEntries != -1 {
		MaxEntries = acf.maxEntries
	}
}

func (acf *AttacherCommandFlags) mergeRetryIntervalStart() {
	if acf.retryIntervalStart != -1*time.Second {
		RetryIntervalStart = acf.retryIntervalStart
	}
}

func (acf *AttacherCommandFlags) mergeRetryIntervalMax() {
	if acf.retryIntervalMax != -1*time.Minute {
		RetryIntervalMax = acf.retryIntervalMax
	}
}

func (acf *AttacherCommandFlags) mergeDefaultFSType() {
	if acf.defaultFSType != "" {
		DefaultFSType = acf.defaultFSType
	}
}

func (acf *AttacherCommandFlags) mergeReconcileSync() {
	if acf.reconcileSync != -1*time.Minute {
		ReconcileSync = acf.reconcileSync
	}
}

func (acf *AttacherCommandFlags) mergeKubeAPIQPS() {
	if acf.kubeAPIQPS != -1 {
		KubeAPIQPS = acf.kubeAPIQPS
	}
}

func (acf *AttacherCommandFlags) mergeKubeAPIBurst() {
	if acf.kubeAPIBurst != -1 {
		KubeAPIBurst = acf.kubeAPIBurst
	}
}

func (acf *AttacherCommandFlags) mergeMaxGPRCLogLength() {
	if acf.maxGRPCLogLength != -1 {
		MaxGRPCLogLength = acf.maxGRPCLogLength
	}
}

func (acf *AttacherCommandFlags) mergeMetricPath() {
	if acf.metricsPath != "" {
		MetricsPath = acf.metricsPath
	}
}
