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
* Processes `VolumeAttach` instances and attaches/detaches volumes. TBD: details + VolumeAttachment finalizers.
* TBD: PV finalizers.
* TBD: async operations - we can't block worker queue for 20 minutes (= current AWS attach timeout).

## Usage

### Dummy mode

Dummy attacher watches for `VolumeAttachment` instances with `Attacher=="csi/dummy"` and marks them attached. It does not use any finalizers and is useful for testing.

To run dummy attacher in `hack/local-up-cluster.sh` environment:

```sh
$ csi-attacher -dummy -kubeconfig=~/kube/config -v 5
```

TBD: running as a pod

### Real attacher

```sh
$ csi-attacher -kubeconfig=~/kube/config -v 5 -csi-address /run/csi/socket
```

TBD: running as a pod

## Vendoring

We use [dep](https://github.com/golang/dep) for management of `vendor/`.

`vendor/k8s.io` is manually copied from `staging/` directory of work-in-progress API for CSI, namely https://github.com/kubernetes/kubernetes/pull/54463.
