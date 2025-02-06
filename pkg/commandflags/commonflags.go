package commandflags

import (
	"flag"
	"time"
)

// common command flags
var (
	Kubeconfig                  string
	CsiAddress                  string
	ShowVersion                 bool
	MetricsAddress              string
	HttpEndpoint                string
	EnableLeaderElection        bool
	LeaderElectionNamespace     string
	LeaderElectionLeaseDuration time.Duration
	LeaderElectionRenewDeadline time.Duration
	LeaderElectionRetryPeriod   time.Duration
)

func InitCommonFlags() {
	initShowVersion()

	initKubeConfig()

	initMetricsAddress()
	initHttpEndpoint()

	initCsiAddress()

	initEnableLeaderElection()
	initLeaderElectionNamespace()
	initLeaderElectionLeaseDuration()
	initLeaderElectionRenewDeadline()
	initLeaderElectionRetryPeriod()
}

func initShowVersion() {
	flag.BoolVar(&ShowVersion, "version", false, "Show version.")
}

func initKubeConfig() {
	flag.StringVar(&Kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file. Required only when running out of cluster.")
}

func initMetricsAddress() {
	// I think we should remove deprecated flag in AIO project, we must change the non-common flags name such as worker-thread to attacher-worker-thread. so it definately be a breaking change
	flag.StringVar(&MetricsAddress, "metrics-address", "", "(deprecated) The TCP network address where the prometheus metrics endpoint will listen (example: `:8080`). The default is empty string, which means metrics endpoint is disabled. Only one of `--metrics-address` and `--http-endpoint` can be set.")
}

func initHttpEndpoint() {
	flag.StringVar(&HttpEndpoint, "http-endpoint", "", "The TCP network address where the HTTP server for diagnostics, including metrics and leader election health check, will listen (example: `:8080`). The default is empty string, which means the server is disabled. Only one of `http-endpoint` and `metrics-address` can be set.")
}

func initCsiAddress() {
	flag.StringVar(&CsiAddress, "csi-address", "/run/csi/socket", "Address of the CSI driver socket.")
}

func initEnableLeaderElection() {
	flag.BoolVar(&EnableLeaderElection, "leader-election", false, "Enable leader election.")
}

func initLeaderElectionNamespace() {
	flag.StringVar(&LeaderElectionNamespace, "leader-election-namespace", "", "Namespace where the leader election resource lives. Defaults to the pod namespace if not set.")
}

func initLeaderElectionLeaseDuration() {
	flag.DurationVar(&LeaderElectionLeaseDuration, "leader-election-lease-duration", 15*time.Second, "Duration, in seconds, that non-leader candidates will wait to force acquire leadership. Defaults to 15 seconds.")
}

func initLeaderElectionRenewDeadline() {
	flag.DurationVar(&LeaderElectionRenewDeadline, "leader-election-renew-deadline", 10*time.Second, "Duration, in seconds, that the acting leader will retry refreshing leadership before giving up. Defaults to 10 seconds.")
}

func initLeaderElectionRetryPeriod() {
	flag.DurationVar(&LeaderElectionRetryPeriod, "leader-election-retry-period", 2*time.Second, "Duration, in seconds, the LeaderElector clients should wait between tries of actions. Defaults to 2 seconds.")
}
