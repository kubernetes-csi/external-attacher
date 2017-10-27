# CSI attacher

The csi-attacher is part of Kubernetes implementation of [Container Storage Interface (CSI)](https://github.com/container-storage-interface/spec).

## Design

In short, it's an external controller that monitors `VolumeAttachment` objects and attaches/detaches volumes to/from nodes. Full design can be found at Kubernetes proposal at https://github.com/kubernetes/community/pull/1258. TODO: update the link after merge.

There is no plan to implement a generic external attacher library, csi-attacher is the only external attacher that exists. If this proves false in future, splitting a generic external-attacher library should be possible with some effort.

## Usage

TBD

## Vendoring

We use [dep](https://github.com/golang/dep) for management of `vendor/`.

`vendor/k8s.io` is manually copied from `staging/` directory of work-in-progress API for CSI, namely https://github.com/kubernetes/kubernetes/pull/54463.
