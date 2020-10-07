# Release notes for v2.2.1
# Changelog since v2.2.0

## Changes by Kind

### Uncategorized

- release-2.2: update release-tools ([#257](https://github.com/kubernetes-csi/external-attacher/pull/257), [@Jiawei0227](https://github.com/Jiawei0227))
  - Build with Go 1.15

## Dependencies

### Added
_Nothing has changed._

### Changed
- github.com/kubernetes-csi/csi-lib-utils: [v0.7.0 → v0.7.1](https://github.com/kubernetes-csi/csi-lib-utils/compare/v0.7.0...v0.7.1)

### Removed
_Nothing has changed._


# Release notes for v2.2.0
# Changelog since v2.1.0

### Bug Fixes

- A bug that prevented the external-attacher from releasing its finalizer from a `PersistentVolume` object that was created using a legacy storage class provisioner and migrated to CSI has been fixed. ([#218](https://github.com/kubernetes-csi/external-attacher/pull/218), [@rfranzke](https://github.com/rfranzke))
- Update package path to v2. Vendoring with dep depends on https://github.com/golang/dep/pull/1963 or the workaround described in v2/README.md. ([#209](https://github.com/kubernetes-csi/external-attacher/pull/209), [@alex1989hu](https://github.com/alex1989hu))


### Other Notable Changes

- Removed usage of annotation csi.volume.kubernetes.io/nodeid on Node objects. The external-attacher now requires Kubernetes 1.14 with feature gate CSINodeInfo enabled. ([#213](https://github.com/kubernetes-csi/external-attacher/pull/213), [@Danil-Grigorev](https://github.com/Danil-Grigorev))


