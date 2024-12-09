# Release notes for v4.0.0

[Documentation](https://kubernetes-csi.github.io)
# Changelog since v3.5.0

## Urgent Upgrade Notes

### (No, really, you MUST read this before you upgrade)

- Change `--default-fstype` from `ext4` to empty string. CSI drivers that depended on ext4 as the default need to change their deployments to explicitly set `--default-fstype=ext4`. ([#358](https://github.com/kubernetes-csi/external-attacher/pull/358), [@Kartik494](https://github.com/Kartik494))

## Changes by Kind

### Uncategorized

- This release update kubernetes module dependencies to v1.25 ([#371](https://github.com/kubernetes-csi/external-attacher/pull/371), [@humblec](https://github.com/humblec))

## Dependencies

### Added
- github.com/emicklei/go-restful/v3: [v3.8.0](https://github.com/emicklei/go-restful/v3/tree/v3.8.0)
- github.com/go-task/slim-sprig: [348f09d](https://github.com/go-task/slim-sprig/tree/348f09d)
- github.com/golang-jwt/jwt/v4: [v4.2.0](https://github.com/golang-jwt/jwt/v4/tree/v4.2.0)
- github.com/golang/snappy: [v0.0.3](https://github.com/golang/snappy/tree/v0.0.3)
- github.com/onsi/ginkgo/v2: [v2.1.4](https://github.com/onsi/ginkgo/v2/tree/v2.1.4)
- google.golang.org/grpc/cmd/protoc-gen-go-grpc: v1.1.0

### Changed
- cloud.google.com/go: v0.81.0 → v0.97.0
- github.com/Azure/go-autorest/autorest/adal: [v0.9.13 → v0.9.20](https://github.com/Azure/go-autorest/autorest/adal/compare/v0.9.13...v0.9.20)
- github.com/Azure/go-autorest/autorest/mocks: [v0.4.1 → v0.4.2](https://github.com/Azure/go-autorest/autorest/mocks/compare/v0.4.1...v0.4.2)
- github.com/Azure/go-autorest/autorest: [v0.11.18 → v0.11.27](https://github.com/Azure/go-autorest/autorest/compare/v0.11.18...v0.11.27)
- github.com/cncf/udpa/go: [5459f2c → 04548b0](https://github.com/cncf/udpa/go/compare/5459f2c...04548b0)
- github.com/cncf/xds/go: [fbca930 → cb28da3](https://github.com/cncf/xds/go/compare/fbca930...cb28da3)
- github.com/envoyproxy/go-control-plane: [63b5d3c → 49ff273](https://github.com/envoyproxy/go-control-plane/compare/63b5d3c...49ff273)
- github.com/go-logr/logr: [v1.2.0 → v1.2.3](https://github.com/go-logr/logr/compare/v1.2.0...v1.2.3)
- github.com/go-logr/zapr: [v1.2.0 → v1.2.3](https://github.com/go-logr/zapr/compare/v1.2.0...v1.2.3)
- github.com/golang/mock: [v1.5.0 → v1.6.0](https://github.com/golang/mock/compare/v1.5.0...v1.6.0)
- github.com/google/go-cmp: [v0.5.5 → v0.5.6](https://github.com/google/go-cmp/compare/v0.5.5...v0.5.6)
- github.com/google/martian/v3: [v3.1.0 → v3.2.1](https://github.com/google/martian/v3/compare/v3.1.0...v3.2.1)
- github.com/google/pprof: [cbba55b → 4bb14d4](https://github.com/google/pprof/compare/cbba55b...4bb14d4)
- github.com/googleapis/gax-go/v2: [v2.0.5 → v2.1.0](https://github.com/googleapis/gax-go/v2/compare/v2.0.5...v2.1.0)
- github.com/matttproud/golang_protobuf_extensions: [c182aff → v1.0.1](https://github.com/matttproud/golang_protobuf_extensions/compare/c182aff...v1.0.1)
- github.com/nxadm/tail: [v1.4.4 → v1.4.8](https://github.com/nxadm/tail/compare/v1.4.4...v1.4.8)
- github.com/onsi/ginkgo: [v1.14.0 → v1.16.4](https://github.com/onsi/ginkgo/compare/v1.14.0...v1.16.4)
- github.com/onsi/gomega: [v1.10.1 → v1.19.0](https://github.com/onsi/gomega/compare/v1.10.1...v1.19.0)
- github.com/yuin/goldmark: [v1.4.1 → v1.4.13](https://github.com/yuin/goldmark/compare/v1.4.1...v1.4.13)
- golang.org/x/crypto: 8634188 → 3147a52
- golang.org/x/mod: 9b9b3d8 → 86c51ed
- golang.org/x/net: cd36cc0 → a158d28
- golang.org/x/sync: 036812b → 886fb93
- golang.org/x/sys: 3681064 → 8c9f86f
- golang.org/x/tools: 897bd77 → v0.1.12
- google.golang.org/api: v0.43.0 → v0.57.0
- google.golang.org/genproto: 42d7afd → c8bf987
- google.golang.org/grpc: v1.40.0 → v1.47.0
- google.golang.org/protobuf: v1.27.1 → v1.28.0
- k8s.io/api: v0.24.0 → v0.25.0
- k8s.io/apimachinery: v0.24.0 → v0.25.0
- k8s.io/client-go: v0.24.0 → v0.25.0
- k8s.io/component-base: v0.24.0 → v0.25.0
- k8s.io/csi-translation-lib: v0.24.0 → v0.25.0
- k8s.io/klog/v2: v2.60.1 → v2.70.1
- k8s.io/kube-openapi: 3ee0da9 → 67bda5d
- k8s.io/utils: 3a6ce19 → ee6ede2
- sigs.k8s.io/json: 9f7c6b3 → f223a00
- sigs.k8s.io/structured-merge-diff/v4: v4.2.1 → v4.2.3

### Removed
- github.com/emicklei/go-restful: [v2.9.5+incompatible](https://github.com/emicklei/go-restful/tree/v2.9.5)
- github.com/form3tech-oss/jwt-go: [v3.2.3+incompatible](https://github.com/form3tech-oss/jwt-go/tree/v3.2.3)
