# Release notes for v4.2.0

[Documentation](https://kubernetes-csi.github.io)

# Changelog since v4.1.1

## Changes by Kind

### Feature

- Logging for gRPC call can now be limited configurably with flag `--max-grpc-log-length` ([#411](https://github.com/kubernetes-csi/external-attacher/pull/411), [@leiyiz](https://github.com/leiyiz))

## Dependencies

### Added
- cloud.google.com/go/compute: v1.12.1

### Changed
- cloud.google.com/go/compute/metadata: v0.2.0 → v0.2.1
- github.com/kubernetes-csi/csi-lib-utils: [v0.12.0 → v0.13.0](https://github.com/kubernetes-csi/csi-lib-utils/compare/v0.12.0...v0.13.0)
- google.golang.org/genproto: 142d8a6 → 1645502
- google.golang.org/grpc: v1.51.0 → v1.52.3
- k8s.io/api: v0.26.0 → v0.26.1
- k8s.io/apimachinery: v0.26.0 → v0.26.1
- k8s.io/client-go: v0.26.0 → v0.26.1
- k8s.io/csi-translation-lib: v0.26.0 → v0.26.1
- k8s.io/klog/v2: v2.80.1 → v2.90.0

### Removed
_Nothing has changed._
