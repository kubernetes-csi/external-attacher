# CSI attacher

The external-attacher is a sidecar container that attaches volumes to nodes by calling `ControllerPublish` and `ControllerUnpublish` functions of CSI drivers. It is necessary because internal Attach/Detach controller running in Kubernetes controller-manager does not have any direct interfaces to CSI drivers.

## Terminology

In Kubernetes, the term *attach* means 3rd party volume attachment to a node. This is common in cloud environments, where the cloud API is able to attach a volume to a node without any code running on the node. In CSI terminology, this corresponds to the `ControllerPublish` call.

*Detach* is the reverse operation, 3rd party volume detachment from a node, `ControllerUnpublish` in CSI terminology.

It is **not** an attach/detach operation performed by a code running on a node, such as an attachment of iSCSI or Fibre Channel volumes. These are typically performed during `NodeStage` and `NodeUnstage` CSI calls and are not done by the external-attacher.

## Overview

The external-attacher is an external controller that monitors `VolumeAttachment` objects created by controller-manager and attaches/detaches volumes to/from nodes (i.e. calls `ControllerPublish`/`ControllerUnpublish`. Full design can be found at Kubernetes proposal at [container-storage-interface.md](https://github.com/kubernetes/design-proposals-archive/blob/main/storage/container-storage-interface.md)

## Compatibility

This information reflects the head of this branch.

| Compatible with CSI Version                                                                | Container Image                     | [Min K8s Version](https://kubernetes-csi.github.io/docs/kubernetes-compatibility.html#minimum-version) | [Recommended K8s Version](https://kubernetes-csi.github.io/docs/kubernetes-compatibility.html#recommended-version) |
| ------------------------------------------------------------------------------------------ | -----------------------------------------| ---- | ---- |
| [CSI Spec v1.5.0](https://github.com/container-storage-interface/spec/releases/tag/v1.5.0) | registry.k8s.io/sig-storage/csi-attacher | 1.17 | 1.22 |

## Feature Status

Various external-attacher releases come with different alpha / beta features.

The following table reflects the head of this branch.

| Feature       | Status  | Default | Description                                                                                   |
| ------------- | ------- | ------- | --------------------------------------------------------------------------------------------- |
| CSIMigration*     | GA      | On      | [Migrating in-tree volume plugins to CSI](https://kubernetes.io/docs/concepts/storage/volumes/#csi-migration). |
| ReadWriteOncePod* | Alpha   | Off     | [Single pod access mode for PersistentVolumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes). |

*) There is no special feature gate for this feature. It is enabled by turning on the corresponding features in Kubernetes.

All other external-attacher features and the external-attacher itself is considered GA and fully supported.

## Usage

It is necessary to create a new service account and give it enough privileges to run the external-attacher, see `deploy/kubernetes/rbac.yaml`. The attacher is then deployed as single Deployment as illustrated below:

```sh
kubectl create deploy/kubernetes/deployment.yaml
```

The external-attacher may run in the same pod with other external CSI controllers such as the external-provisioner, external-snapshotter and/or external-resizer.

Note that the external-attacher does not scale with more replicas. Only one external-attacher is elected as leader and running. The others are waiting for the leader to die. They re-elect a new active leader in ~15 seconds after death of the old leader.

### Command line options

#### Important optional arguments that are highly recommended to be used

* `--csi-address <path to CSI socket>`: This is the path to the CSI driver socket inside the pod that the external-attacher container will use to issue CSI operations (`/run/csi/socket` is used by default).

* `--leader-election`: Enables leader election. This is useful when there are multiple replicas of the same external-attacher running for one CSI driver. Only one of them may be active (=leader). A new leader will be re-elected when current leader dies or becomes unresponsive for ~15 seconds.

* `--leader-election-namespace <namespace>`: Namespace where the external-attacher runs and where leader election object will be created. It is recommended that this parameter is populated from Kubernetes DownwardAPI.

* `--timeout <duration>`: Timeout of all calls to CSI driver. It should be set to value that accommodates majority of `ControllerPublish` and `ControllerUnpublish` calls. See [CSI error and timeout handling](#csi-error-and-timeout-handling) for details. 15 seconds is used by default.

* `--worker-threads`: The number of goroutines for processing VolumeAttachments. 10 workers is used by default.

* `--max-entries`: The max number of entries per page for processing ListVolumes. 0 means no limit and it is the default value.

* `--retry-interval-start`: The exponential backoff for failures. See [CSI error and timeout handling](#csi-error-and-timeout-handling) for details. 1 second is used by default.

* `--retry-interval-max`: The exponential backoff maximum value. See [CSI error and timeout handling](#csi-error-and-timeout-handling) for details. 5 minutes is used by default.

* `--http-endpoint`: The TCP network address where the HTTP server for diagnostics, including metrics and leader election health check, will listen (example: `:8080` which corresponds to port 8080 on local host). The default is empty string, which means the server is disabled.

* `--metrics-path`: The HTTP path where prometheus metrics will be exposed. Default is `/metrics`.

* `--reconcile-sync`: Resync frequency of the attached volumes with the driver. See [Periodic re-sync](#periodic-re-sync) for details. 1 minute is used by default.

* `--kube-api-qps`: The number of requests per second sent by a Kubernetes client to the Kubernetes API server. Defaults to `5.0`.

* `--kube-api-burst`: The number of requests to the Kubernetes API server, exceeding the QPS, that can be sent at any given time. Defaults to `10`.

* `--leader-election-lease-duration <duration>`: Duration, in seconds, that non-leader candidates will wait to force acquire leadership. Defaults to 15 seconds.

* `--leader-election-renew-deadline <duration>`: Duration, in seconds, that the acting leader will retry refreshing leadership before giving up. Defaults to 10 seconds.

* `--leader-election-retry-period <duration>`: Duration, in seconds, the LeaderElector clients should wait between tries of actions. Defaults to 5 seconds.

* `--default-fstype <type>`: The default filesystem type of the volume to publish. Defaults to empty string.

#### Other recognized arguments

* `--kubeconfig <path>`: Path to Kubernetes client configuration that the external-attacher uses to connect to Kubernetes API server. When omitted, default token provided by Kubernetes will be used. This option is useful only when the external-attacher does not run as a Kubernetes pod, e.g. for debugging.

* `--metrics-address`: (deprecated) The TCP network address where the prometheus metrics endpoint and leader election health check will run (example: `:8080` which corresponds to port 8080 on local host). The default is empty string, which means metrics and leader election check endpoint is disabled.

* `--resync <duration>`: Internal resync interval when the external-attacher re-evaluates all existing `VolumeAttachment` instances and tries to fulfill them, i.e. attach / detach corresponding volumes. It does not affect re-tries of failed CSI calls! It should be used only when there is a bug in Kubernetes watch logic.

* `--automaxprocs`: Automatically set the `GOMAXPROCS` environment variable to match the configured Linux container CPU quota. Defaults to false.

* `--version`: Prints current external-attacher version and quits.

* All glog / klog arguments are supported, such as `-v <log level>` or `-alsologtostderr`.

### CSI error and timeout handling

The external-attacher invokes all gRPC calls to CSI driver with timeout provided by `--timeout` command line argument (15 seconds by default).

* `ControllerPublish`: The call might have timed out just before the driver attached a volume and was sending a response. From that reason, timeouts from `ControllerPublish` is considered as "*volume may be attached*" or "*volume is being attached in the background*." The external-attacher will re-try calling `ControllerPublish` after exponential backoff until it gets either successful response or final (non-timeout) error that the volume cannot be attached.
* `ControllerUnpublish`: This is similar to `ControllerPublish`, The external-attacher will re-try calling `ControllerUnpublish` with exponential backoff after timeout until it gets either successful response or a final error that the volume cannot be detached.
* `Probe`: The external-attacher re-tries calling Probe until the driver reports it's ready. It re-tries also when it receives timeout from `Probe` call. The external-attacher has no limit of retries. It is expected that ReadinessProbe on the driver container will catch case when the driver takes too long time to get ready.
* `GetPluginInfo`, `GetPluginCapabilitiesRequest`, `ControllerGetCapabilities`: The external-attacher expects that these calls are quick and does not retry them on any error, including timeout. Instead, it assumes that the driver is faulty and exits. Note that Kubernetes will likely start a new attacher container and it will start with `Probe` call.

Correct timeout value depends on the storage backend and how quickly it is able to processes `ControllerPublish` and `ControllerUnpublish` calls. The value should be set to accommodate majority of them. It is fine if some calls time out - such calls will be re-tried after exponential backoff (starting with `--retry-interval-start`), however, this backoff will introduce delay when the call times out several times for a single volume (up to `--retry-interval-max`).

### Periodic re-sync

When CSI driver supports `LIST_VOLUMES` and `LIST_VOLUMES_PUBLISHED_NODES` capabilities, the external attacher periodically syncs volume attachments requested by Kubernetes with the actual state reported by CSI driver. Volumes detached by any 3rd party, but still required to be attached by Kubernetes, will be re-attached back. Frequency of this re-sync is controlled by `--reconcile-sync` command line parameter.

### HTTP endpoint

The external-attacher optionally exposes an HTTP endpoint at address:port specified by `--http-endpoint` argument. When set, these two paths are exposed:

* Metrics path, as set by `--metrics-path` argument (default is `/metrics`).
* Leader election health check at `/healthz/leader-election`. It is recommended to run a liveness probe against this endpoint when leader election is used to kill external-attacher leader that fails to connect to the API server to renew its leadership. See https://github.com/kubernetes-csi/csi-lib-utils/issues/66 for details.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

* Slack channels
  * [#wg-csi](https://kubernetes.slack.com/messages/wg-csi)
  * [#sig-storage](https://kubernetes.slack.com/messages/sig-storage)
* [Mailing list](https://groups.google.com/forum/#!forum/kubernetes-sig-storage)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
