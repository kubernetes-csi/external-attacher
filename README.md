[![Build Status](https://travis-ci.org/kubernetes-csi/external-attacher.svg?branch=master)](https://travis-ci.org/kubernetes-csi/external-attacher)

# CSI attacher

The external-attacher is a sidecar container that attaches volumes to nodes by calling `ControllerPublish` and `ControlerUnpublish` functions of CSI drivers. It is necessary because internal Attach/Detach controller running in Kubernetes controller-manager does not have any direct interfaces to CSI drivers.

## Terminology

In Kubernetes, the term *attach* means 3rd party volume attachment to a node. This is common in cloud environments, where the cloud API is able to attach a volume to a node without any code running on the node. In CSI terminology, this corresponds to the `ControllerPublish` call.

*Detach* is the reverse operation, 3rd party volume detachment from a node, `ControllerUnpublish` in CSI terminology.

It is **not** an attach/detach operation performed by a code running on a node, such as an attachment of iSCSI or Fibre Channel volumes. These are typically performed during `NodeStage` and `NodeUnstage` CSI calls and are not done by the external-attacher.

## Overview
The external-attacher is an external controller that monitors `VolumeAttachment` objects created by controller-manager and attaches/detaches volumes to/from nodes (i.e. calls `ControllerPublish`/`ControllerUnpublish`. Full design can be found at Kubernetes proposal at [container-storage-interface.md](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/container-storage-interface.md)

## Compatibility

This information reflects the head of this branch.

| Compatible with CSI Version                                                                | Container Image             | Min K8s Version |
| ------------------------------------------------------------------------------------------ | ----------------------------| --------------- |
| [CSI Spec v1.0.0](https://github.com/container-storage-interface/spec/releases/tag/v1.0.0) | quay.io/k8scsi/csi-attacher | 1.15            |

## Feature Status

Various external-attacher releases come with different alpha / beta features.

The following table reflects the head of this branch.

| Feature       | Status  | Default | Description                                                                                   |
| ------------- | ------- | ------- | --------------------------------------------------------------------------------------------- |
| CSINode*      | Beta    | On      | external-attacher uses the CSINode object to get the driver's node name instead of the Node annotation. |
| CSIMigration* | Alpha   | On      | [Migrating in-tree volume plugins to CSI](https://kubernetes.io/docs/concepts/storage/volumes/#csi-migration). |

*) There are no special feature gates for these features. They are enabled by turning on the corresponding features in Kubernetes.

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

* `--retry-interval-start`: The exponential backoff for failures. See [CSI error and timeout handling](#csi-error-and-timeout-handling) for details. 1 second is used by default.

* `--retry-interval-end`: The exponential backoff maximum value. See [CSI error and timeout handling](#csi-error-and-timeout-handling) for details. 5 minutes is used by default.

#### Other recognized arguments
* `--dummy`: Runs the external-attacher in dummy mode, i.e. without any CSI driver. All volumes are immediately reported as attached / detached as controller-manager requires. This option can be used for debugging of other CSI components such as Kubernetes Attach / Detach controller.

* `--kubeconfig <path>`: Path to Kubernetes client configuration that the external-attacher uses to connect to Kubernetes API server. When omitted, default token provided by Kubernetes will be used. This option is useful only when the external-attacher does not run as a Kubernetes pod, e.g. for debugging.

* `--resync <duration>`: Internal resync interval when the external-attacher re-evaluates all existing `VolumeAttachment` instances and tries to fulfill them, i.e. attach / detach corresponding volumes. It does not affect re-tries of failed CSI calls! It should be used only when there is a bug in Kubernetes watch logic.

* `--version`: Prints current external-attacher version and quits.

* All glog / klog arguments are supported, such as `-v <log level>` or `-alsologtostderr`.

### CSI error and timeout handling
The external-attacher invokes all gRPC calls to CSI driver with timeout provided by `--timeout` command line argument (15 seconds by default).

* `ControllerPublish`: The call might have timed out just before the driver attached a volume and was sending a response. From that reason, timeouts from `ControllerPublish` is considered as "*volume may be attached*" or "*volume is being attached in the background*." The external-attacher will re-try calling `ControllerPublish` after exponential backoff until it gets either successful response or final (non-timeout) error that the volume cannot be attached.
* `ControllerUnpublish`: This is similar to `ControllerPublish`, The external-attacher will re-try calling `ControllerUnpublish` with exponential backoff after timeout until it gets either successful response or a final error that the volume cannot be detached.
* `Probe`: The external-attacher re-tries calling Probe until the driver reports it's ready. It re-tries also when it receives timeout from `Probe` call. The external-attacher has no limit of retries. It is expected that ReadinessProbe on the driver container will catch case when the driver takes too long time to get ready.
* `GetPluginInfo`, `GetPluginCapabilitiesRequest`, `ControllerGetCapabilities`: The external-attacher expects that these calls are quick and does not retry them on any error, including timeout. Instead, it assumes that the driver is faulty and exits. Note that Kubernetes will likely start a new attacher container and it will start with `Probe` call.

Correct timeout value depends on the storage backend and how quickly it is able to processes `ControllerPublish` and `ControllerUnpublish` calls. The value should be set to accommodate majority of them. It is fine if some calls time out - such calls will be re-tried after exponential backoff (starting with `--retry-interval-start`), however, this backoff will introduce delay when the call times out several times for a single volume (up to `--retry-interval-max`).

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

* Slack channels
  * [#wg-csi](https://kubernetes.slack.com/messages/wg-csi)
  * [#sig-storage](https://kubernetes.slack.com/messages/sig-storage)
* [Mailing list](https://groups.google.com/forum/#!forum/kubernetes-sig-storage)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
