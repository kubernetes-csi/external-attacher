# Changelog since v2.0.0

The attacher now supports CSI version 1.2.0, namely LIST_VOLUMES_PUBLISHED_NODES capability. When the capability is supported by CSI driver, the attacher periodically syncs volume attachments requested by Kubernetes with actual state reported by CSI driver.

## New Features

- The attacher reconciles VolumeAttachment status with actual back-end volume attachment state if plugin supports LIST_VOLUMES_PUBLISHED_NODES capability. ([#184](https://github.com/kubernetes-csi/external-attacher/pull/184), [@davidz627](https://github.com/davidz627))
- Add prometheus metrics to CSI external-attacher under the /metrics endpoint. This can be enabled via the "--metrics-address" and "--metrics-path" options. ([#201](https://github.com/kubernetes-csi/external-attacher/pull/201), [@saad-ali](https://github.com/saad-ali))


## Other Notable Changes

- Migrated to Go modules, so the source builds also outside of GOPATH. ([#188](https://github.com/kubernetes-csi/external-attacher/pull/188), [@pohly](https://github.com/pohly))
