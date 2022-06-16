# Release notes for v3.2.0

[Documentation](https://kubernetes-csi.github.io)
# Changelog since v3.1.0

## Urgent Upgrade Notes

### (No, really, you MUST read this before you upgrade)

- For drivers that support CSI Migration, a "migrated" label was added to the csi_sidecar_operations_seconds metric that indicates if the call is from the migration path. Metric collectors should be updated with the new field. ([#292](https://github.com/kubernetes-csi/external-attacher/pull/292), [@nearora-msft](https://github.com/nearora-msft))

## Changes by Kind

### Feature

- Add configurable throughput parameters for clients to API server ([#286](https://github.com/kubernetes-csi/external-attacher/pull/286), [@RaunakShah](https://github.com/RaunakShah))
- Bump csi translation lib version ([#293](https://github.com/kubernetes-csi/external-attacher/pull/293), [@andyzhangx](https://github.com/andyzhangx))

### Bug or Regression

- Fix a bug where external-attacher finalizer not get lifted for migrated PV when CSIMigration is turned off. ([#294](https://github.com/kubernetes-csi/external-attacher/pull/294), [@Jiawei0227](https://github.com/Jiawei0227))
- Fixed volume detach when CSI driver is missing in CSINode object. ([#299](https://github.com/kubernetes-csi/external-attacher/pull/299), [@jackkleeman](https://github.com/jackkleeman))

### Uncategorized

- Updated runtime (Go 1.16) and dependencies ([#295](https://github.com/kubernetes-csi/external-attacher/pull/295), [@pohly](https://github.com/pohly))

## Dependencies

### Added
- github.com/moby/spdystream: [v0.2.0](https://github.com/moby/spdystream/tree/v0.2.0)
- github.com/niemeyer/pretty: [a10e7ca](https://github.com/niemeyer/pretty/tree/a10e7ca)

### Changed
- github.com/Azure/go-autorest/autorest: [v0.11.1 → v0.11.12](https://github.com/Azure/go-autorest/autorest/compare/v0.11.1...v0.11.12)
- github.com/cncf/udpa/go: [efcf912 → 5459f2c](https://github.com/cncf/udpa/go/compare/efcf912...5459f2c)
- github.com/container-storage-interface/spec: [v1.3.0 → v1.4.0](https://github.com/container-storage-interface/spec/compare/v1.3.0...v1.4.0)
- github.com/creack/pty: [v1.1.7 → v1.1.11](https://github.com/creack/pty/compare/v1.1.7...v1.1.11)
- github.com/envoyproxy/go-control-plane: [v0.9.7 → fd9021f](https://github.com/envoyproxy/go-control-plane/compare/v0.9.7...fd9021f)
- github.com/fsnotify/fsnotify: [v1.4.9 → v1.4.7](https://github.com/fsnotify/fsnotify/compare/v1.4.9...v1.4.7)
- github.com/go-logr/logr: [v0.3.0 → v0.4.0](https://github.com/go-logr/logr/compare/v0.3.0...v0.4.0)
- github.com/gogo/protobuf: [v1.3.1 → v1.3.2](https://github.com/gogo/protobuf/compare/v1.3.1...v1.3.2)
- github.com/golang/protobuf: [v1.4.3 → v1.5.1](https://github.com/golang/protobuf/compare/v1.4.3...v1.5.1)
- github.com/google/go-cmp: [v0.5.4 → v0.5.5](https://github.com/google/go-cmp/compare/v0.5.4...v0.5.5)
- github.com/googleapis/gnostic: [v0.5.3 → v0.5.4](https://github.com/googleapis/gnostic/compare/v0.5.3...v0.5.4)
- github.com/gorilla/websocket: [4201258 → v1.4.2](https://github.com/gorilla/websocket/compare/4201258...v1.4.2)
- github.com/imdario/mergo: [v0.3.11 → v0.3.12](https://github.com/imdario/mergo/compare/v0.3.11...v0.3.12)
- github.com/kisielk/errcheck: [v1.2.0 → v1.5.0](https://github.com/kisielk/errcheck/compare/v1.2.0...v1.5.0)
- github.com/kr/text: [v0.1.0 → v0.2.0](https://github.com/kr/text/compare/v0.1.0...v0.2.0)
- github.com/kubernetes-csi/csi-lib-utils: [v0.9.0 → v0.9.1](https://github.com/kubernetes-csi/csi-lib-utils/compare/v0.9.0...v0.9.1)
- github.com/moby/term: [672ec06 → df9cb8a](https://github.com/moby/term/compare/672ec06...df9cb8a)
- github.com/prometheus/client_golang: [v1.8.0 → v1.9.0](https://github.com/prometheus/client_golang/compare/v1.8.0...v1.9.0)
- github.com/prometheus/common: [v0.15.0 → v0.19.0](https://github.com/prometheus/common/compare/v0.15.0...v0.19.0)
- github.com/prometheus/procfs: [v0.2.0 → v0.6.0](https://github.com/prometheus/procfs/compare/v0.2.0...v0.6.0)
- github.com/yuin/goldmark: [v1.1.32 → v1.2.1](https://github.com/yuin/goldmark/compare/v1.1.32...v1.2.1)
- golang.org/x/crypto: 5f87f34 → 5ea612d
- golang.org/x/net: 986b41b → afb366f
- golang.org/x/oauth2: 08078c5 → cd4f82c
- golang.org/x/sync: 6e8e738 → 09787c9
- golang.org/x/sys: f9fddec → 4fbd30e
- golang.org/x/term: 2321bbc → de623e6
- golang.org/x/text: v0.3.4 → v0.3.6
- golang.org/x/time: 7e3f01d → f8bda1e
- golang.org/x/tools: b303f43 → 113979e
- google.golang.org/genproto: 8c77b98 → 75c7a85
- google.golang.org/grpc: v1.34.0 → v1.36.0
- google.golang.org/protobuf: v1.25.0 → v1.26.0
- gopkg.in/check.v1: 41f04d3 → 8fa4692
- gopkg.in/yaml.v3: eeeca48 → 496545a
- gotest.tools/v3: v3.0.2 → v3.0.3
- k8s.io/api: v0.20.0 → v0.21.0
- k8s.io/apimachinery: v0.20.0 → v0.21.0
- k8s.io/client-go: v0.20.0 → v0.21.0
- k8s.io/component-base: v0.20.0 → v0.21.0
- k8s.io/csi-translation-lib: v0.20.0 → v0.21.0
- k8s.io/klog/v2: v2.4.0 → v2.8.0
- k8s.io/kube-openapi: d219536 → f622666
- k8s.io/utils: 67b214c → 2afb431
- sigs.k8s.io/structured-merge-diff/v4: v4.0.2 → v4.1.1

### Removed
- github.com/docker/spdystream: [449fdfc](https://github.com/docker/spdystream/tree/449fdfc)
- gotest.tools: v2.2.0+incompatible
