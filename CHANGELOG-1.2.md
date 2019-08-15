# Changelog since v1.2.0

## Bug Fixes

- Fixed handling of ControllerUnpublish errors. The attacher will retry to ControllerUnpublish a volume after any error except for NotFound. ([#168](https://github.com/kubernetes-csi/external-attacher/pull/168), [@jsafrane](https://github.com/jsafrane))

# Changelog since v1.1.0

## New Features

- Adds CSI Migration support for Azure Disk/File, fixes for backward compatible AccessModes for GCE PD. ([#156](https://github.com/kubernetes-csi/external-attacher/pull/156), [@davidz627](https://github.com/davidz627))
- Support attachment of inline volumes migrated to CSI ([#154](https://github.com/kubernetes-csi/external-attacher/pull/154), [@ddebroy](https://github.com/ddebroy))
- Adds --retry-interval-max and --retry-interval-start to the csi-attacher parameters to allow users to limit the exponential backoff retry time for requests. ([#141](https://github.com/kubernetes-csi/external-attacher/pull/141), [@barp](https://github.com/barp))


## Bug Fixes

- The default leader election type will be `configmaps` if not specified in the command line ([#144](https://github.com/kubernetes-csi/external-attacher/pull/144), [@mlmhl](https://github.com/mlmhl))
