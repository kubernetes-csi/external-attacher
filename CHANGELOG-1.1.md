# Changelog since v1.0.1

## Deprecations

* Command line flag `-connection-timeout` is deprecated and has no effect.
* Command line flag `--leader-election-identity` is deprecated and has no effect.
* Command line flag `--leader-election-type` is deprecated. Support for Configmaps-based
  leader election will be removed in the future in favor of Lease-based leader election.
  The default currently remains as `configmaps` for backwards compatibility.

## Notable Features

* The external attacher now tries to connect to CSI driver indefinitely. ([#123](https://github.com/kubernetes-csi/external-attacher/pull/123))

* The external attacher uses CSINode API from `storage.k8s.io/v1beta1`. Handling of alpha `CSINodeInfo` objects was removed. ([#134](https://github.com/kubernetes-csi/external-attacher/pull/134))

* [In-tree storage plugin to CSI Driver Migration](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20190129-csi-migration.md) is now alpha. ([#117](https://github.com/kubernetes-csi/external-attacher/pull/117))

* README.md has been significantly enhanced. ([#130](https://github.com/kubernetes-csi/external-attacher/pull/130))

* Add support for Lease based leader election. Enable this by setting `--leader-election-type=leases` ([#135](https://github.com/kubernetes-csi/external-attacher/pull/135))

## Other notable changes

* Update vendor to bring in updated CSI translation library ([#133](https://github.com/kubernetes-csi/external-attacher/pull/133))
* Use distroless as base image ([#132](https://github.com/kubernetes-csi/external-attacher/pull/132))
* Refactor external-attacher to use csi-lib-utils/rpc ([#127](https://github.com/kubernetes-csi/external-attacher/pull/127))
* Fix #128 - Cannot attach raw block volumes ([#129](https://github.com/kubernetes-csi/external-attacher/pull/129))
* Migrate to k8s.io/klog from glog. ([#119](https://github.com/kubernetes-csi/external-attacher/pull/119))
* Update deployment.yaml ([#121](https://github.com/kubernetes-csi/external-attacher/pull/121))
* Correct markdown linter errors. ([#116](https://github.com/kubernetes-csi/external-attacher/pull/116))
* Add function details in external attacher exported functions. ([#115](https://github.com/kubernetes-csi/external-attacher/pull/115))
* Remove explicit `nil` error initialization. ([#114](https://github.com/kubernetes-csi/external-attacher/pull/114))
* Skip processing of Attach/DetachError changes ([#104](https://github.com/kubernetes-csi/external-attacher/pull/104))
* Update CSINodeInfo ([#101](https://github.com/kubernetes-csi/external-attacher/pull/101))
* Implement PUBLISH_READONLY capability ([#98](https://github.com/kubernetes-csi/external-attacher/pull/98))
