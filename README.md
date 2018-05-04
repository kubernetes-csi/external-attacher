[![Build Status](https://travis-ci.org/kubernetes-csi/external-attacher.svg?branch=master)](https://travis-ci.org/kubernetes-csi/external-attacher)
# CSI attacher

The csi-attacher is part of Kubernetes implementation of [Container Storage Interface (CSI)](https://github.com/container-storage-interface/spec).

## Overview

In short, it's an external controller that monitors `VolumeAttachment` objects and attaches/detaches volumes to/from nodes. Full design can be found at Kubernetes proposal at https://github.com/kubernetes/community/pull/1258. TODO: update the link after merge.

There is no plan to implement a generic external attacher library, csi-attacher is the only external attacher that exists. If this proves false in future, splitting a generic external-attacher library should be possible with some effort.

## Design

External attacher follows [controller](https://github.com/kubernetes/community/blob/master/contributors/devel/controllers.md) pattern and uses informers to watch for `VolumeAttachment` and `PersistentVolume` create/update/delete events. It filters out `VolumeAttachment` instances with `Attacher==<CSI driver name>` and processes these events in workqueues with exponential backoff. Real handling is deferred to `Handler` interface.

`Handler` interface has two implementations, trivial and real one.

### Trivial handler

Trivial handler will be used for CSI drivers that don't support `ControllerPublish` calls and marks all `VolumeAttachment` as attached. It does not use any finalizers. This attacher can also be used for testing.

### Real attacher

"Real" attacher talks to CSI over socket (`/run/csi/socket` by default, configurable by `-csi-address`). The attacher tries to connect for `-connection-timeout` (1 minute by default), allowing CSI driver to start and create its server socket a bit later.

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

As consequence, the attacher must be available until all PVs that refer to the CSI driver *are removed*. Even fully detached PVs have attacher's finalizer that is removed only after the PV is marked for deletion.

#### Alternatives considered

Secondary cache and locks was considered to keep a map PV -> list of VolumeAttachments that use the PV. Attacher's finalizer could be removed from a PV immediately after the last VolumeAttachment was deleted. Keeping this map is either racy or requires long critical sections with complicated error recovery.

## Usage

### Dummy mode

Dummy attacher watches for `VolumeAttachment` instances with `Attacher=="csi/dummy"` and marks them attached. It does not use any finalizers and is useful for testing.

To run dummy attacher in `hack/local-up-cluster.sh` environment:

```sh
$ csi-attacher -dummy -kubeconfig ~/.kube/config -v 5
```

### Real attacher

#### Running on command line
For debugging, it's possible to run the attacher on command line:

```sh
$ csi-attacher -kubeconfig ~/.kube/config -v 5 -csi-address /run/csi/socket
```

#### Running in a deployment
It is necessary to create a new service account and give it enough privileges to run the attacher. We provide one omnipotent yaml file that creates everything that's necessary, however it should be split into multiple files in production.

```sh
$ kubectl create deploy/kubernetes/deployment.yaml
```

Note that the attacher does not scale with more replicas. Only one attacher is elected as leader and running. The others are waiting for the leader to die. They re-elect a new active leader in ~15 seconds after death of the old leader.

## Vendoring

We use [dep](https://github.com/golang/dep) for management of `vendor/`.

`vendor/k8s.io` is manually copied from `staging/` directory of work-in-progress API for CSI, namely https://github.com/kubernetes/kubernetes/pull/54463.
