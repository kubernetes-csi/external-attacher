## Running on command line

For debugging, it's possible to run the external-attacher on command line:

```sh
csi-attacher -kubeconfig ~/.kube/config -v 5 -csi-address /run/csi/socket
```

## Implementation details

The external-attacher follows [controller](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md) pattern and uses informers to watch for `VolumeAttachment` and `PersistentVolume` create/update/delete events. It filters out `VolumeAttachment` instances with `Attacher==<CSI driver name>` and processes these events in workqueues with exponential backoff. Real handling is deferred to `Handler` interface.

`Handler` interface has two implementations, trivial and real one.

### Trivial handler

Trivial handler will be used for CSI drivers that don't support `ControllerPublish` calls and marks all `VolumeAttachment` as attached. It does not use any finalizers. This attacher can also be used for testing.

### Real attacher

"Real" attacher talks to CSI over socket (`/run/csi/socket` by default, configurable by `-csi-address`).

The attacher then:

* Discovers the supported attacher name by `GetPluginInfo` calls. The attacher only processes `VolumeAttachment` instances that have `Attacher==GetPluginInfoResponse.Name`.
* Uses `ControllerGetCapabilities` to find out if CSI driver supports `ControllerPublish` calls. It degrades to trivial mode if not.
* Processes new/updated `VolumeAttachment` instances and attaches/detaches volumes:
  * `VolumeAttachment` without `DeletionTimestamp`:
    * Ignore `VolumeAttachment` that wants to attach PV with `DeletionTimestamp`.
    * A finalizer is added to `VolumeAttachment` instance to preserve the object after deletion so we can detach the volume.
    * A finalizer is added to referenced PV instance to preserve the PV. Attacher needs information from the PV to detach the volume.
    * CSI `ControllerPublishVolume` is called.
    * `AttachmentMetadata` is saved to `VolumeAttachment`.
    * On any error, the `VolumeAttachment` is re-queued with exponential backoff.
  * `VolumeAttachment` with `DeletionTimestamp`:
    * CSI `ControllerUnpublishVolume` is called.
    * A finalizer is removed from `VolumeAttachment`. At this point, the API server is going to delete this instance and "deleted `VolumeAttachment`" event will be received.

* Processes deleted `VolumeAttachment` instances:
  * Pokes PV queue with name of the detached PV. This triggers removal of finalizer on PV, if needed.

* Processes added/updated PV to remove finalizer on PVs:
  * Ignore PVs that don't have DeletionTimestamp.
  * Checks that the PV is not used by any `VolumeAttachment` instance.
  * Removes Attacher's finalizer on the PV if so.
  * On any error, the PV is re-queued with exponential backoff.

#### Concurrency

Both PV queue and `VolumeAttachment` queue run in parallel. To ensure that removal of PV finalizers work without races:

* The controller attaches PVs only when the PV has no DeletionTimestamp and has attacher's finalizer.
* The controller removes finalizer only from PVs that have DeletionTimestamp.

As consequence, the external-attacher must be available until all PVs that refer to the CSI driver *are removed*. Even fully detached PVs have external-attacher's finalizer that is removed only after the PV is marked for deletion.

#### Alternatives considered

Secondary cache and locks was considered to keep a map PV -> list of VolumeAttachments that use the PV. Attacher's finalizer could be removed from a PV immediately after the last VolumeAttachment was deleted. Keeping this map is either racy or requires long critical sections with complicated error recovery.

## Vendoring

We use [dep](https://github.com/golang/dep) for management of `vendor/`.

`vendor/k8s.io` is manually copied from `staging/` directory of work-in-progress API for CSI, namely <https://github.com/kubernetes/kubernetes/pull/54463>.
