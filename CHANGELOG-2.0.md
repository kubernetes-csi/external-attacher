# Changelog since v1.2.0

The version 2.0 is not compatible with v1.x. See Action Required section and update CSI driver manifests.

## Action Required

- The external-attacher now uses PATCH HTTP method to update API objects. Please update the attacher RBAC policy to allow the attacher to `patch`  VolumeAttachments and PersistentVolumes. See `deploy/kubernetes/rbac.yaml` for an example. ([#177](https://github.com/kubernetes-csi/external-attacher/pull/177), [@jsafrane](https://github.com/jsafrane))
- The `-connection-timeout`, `-leader-election-type` and `-leader-election-identity` flags, deprecated in v1.2, have been removed. Please update your manifests for the external-attacher. Leader election uses `lease` object now. Rolling update from v1.2.y release may not work, as multiple leaders may be elected during the update (one using config maps and another using `lease` object).
- The `-dummy` flag has been removed. Please update your manifests for the external-attacher. ([#173](https://github.com/kubernetes-csi/external-attacher/pull/173), [@jsafrane](https://github.com/jsafrane))
- Processing of ControllerUnpublish errors has changed. CSI drivers SHALL return success (0), when a deleted node or volume implies that the volume is detached from the node. The external attacher treats NotFound error as any other error and it assumes that the volume may still be attached to the node. Please check behavior of your CSI driver and fix it accordingly. ([#165](https://github.com/kubernetes-csi/external-attacher/pull/165), [@jsafrane](https://github.com/jsafrane))


## Bug Fixes

- Fixed issue to actually translate backwards compatible access modes for CSI Migration ([#163](https://github.com/kubernetes-csi/external-attacher/pull/163), [@davidz627](https://github.com/davidz627))
- The external attacher now exits when it loses the connection to a CSI driver. This speeds up re-election of a new attacher leader that has connection to the driver. ([182](https://github.com/kubernetes-csi/external-attacher/pull/182), [@jsafrane](https://github.com/jsafrane))


## Other Notable Changes

- Added a new flag `--worker-threads` to control the number of goroutines for processing VolumeAttachments. The default value is 10 workers. ([#175](https://github.com/kubernetes-csi/external-attacher/pull/175), [@hoyho](https://github.com/hoyho))
