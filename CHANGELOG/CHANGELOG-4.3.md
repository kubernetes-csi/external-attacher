# Release notes for v4.3.0

[Documentation](https://kubernetes-csi.github.io)

# Changelog since v4.2.0

## Changes by Kind

### Bug or Regression

- Fix: CVE-2022-41723 ([#415](https://github.com/kubernetes-csi/external-attacher/pull/415), [@andyzhangx](https://github.com/andyzhangx))

## Dependencies

### Added
- cloud.google.com/go/errorreporting: v0.3.0
- cloud.google.com/go/firestore: v1.9.0
- cloud.google.com/go/logging: v1.6.1
- cloud.google.com/go/maps: v0.1.0
- cloud.google.com/go/pubsublite: v1.5.0
- cloud.google.com/go/spanner: v1.41.0
- cloud.google.com/go/vmwareengine: v0.1.0
- github.com/creack/pty: [v1.1.9](https://github.com/creack/pty/tree/v1.1.9)

### Changed
- cloud.google.com/go/aiplatform: v1.24.0 → v1.27.0
- cloud.google.com/go/bigquery: v1.43.0 → v1.44.0
- cloud.google.com/go/compute/metadata: v0.2.1 → v0.2.3
- cloud.google.com/go/compute: v1.12.1 → v1.15.1
- cloud.google.com/go/datastore: v1.1.0 → v1.10.0
- cloud.google.com/go/iam: v0.7.0 → v0.8.0
- cloud.google.com/go/pubsub: v1.3.1 → v1.27.1
- github.com/census-instrumentation/opencensus-proto: [v0.2.1 → v0.4.1](https://github.com/census-instrumentation/opencensus-proto/compare/v0.2.1...v0.4.1)
- github.com/cespare/xxhash/v2: [v2.1.2 → v2.2.0](https://github.com/cespare/xxhash/v2/compare/v2.1.2...v2.2.0)
- github.com/cncf/udpa/go: [04548b0 → c52dc94](https://github.com/cncf/udpa/go/compare/04548b0...c52dc94)
- github.com/cncf/xds/go: [cb28da3 → 06c439d](https://github.com/cncf/xds/go/compare/cb28da3...06c439d)
- github.com/container-storage-interface/spec: [v1.7.0 → v1.8.0](https://github.com/container-storage-interface/spec/compare/v1.7.0...v1.8.0)
- github.com/envoyproxy/go-control-plane: [49ff273 → v0.10.3](https://github.com/envoyproxy/go-control-plane/compare/49ff273...v0.10.3)
- github.com/envoyproxy/protoc-gen-validate: [v0.1.0 → v0.9.1](https://github.com/envoyproxy/protoc-gen-validate/compare/v0.1.0...v0.9.1)
- github.com/go-openapi/jsonpointer: [v0.19.5 → v0.19.6](https://github.com/go-openapi/jsonpointer/compare/v0.19.5...v0.19.6)
- github.com/go-openapi/jsonreference: [v0.20.0 → v0.20.1](https://github.com/go-openapi/jsonreference/compare/v0.20.0...v0.20.1)
- github.com/golang/glog: [23def4e → v1.0.0](https://github.com/golang/glog/compare/23def4e...v1.0.0)
- github.com/golang/protobuf: [v1.5.2 → v1.5.3](https://github.com/golang/protobuf/compare/v1.5.2...v1.5.3)
- github.com/google/pprof: [94a9f03 → 4bb14d4](https://github.com/google/pprof/compare/94a9f03...4bb14d4)
- github.com/kr/pretty: [v0.2.0 → v0.3.0](https://github.com/kr/pretty/compare/v0.2.0...v0.3.0)
- github.com/onsi/ginkgo/v2: [v2.4.0 → v2.9.1](https://github.com/onsi/ginkgo/v2/compare/v2.4.0...v2.9.1)
- github.com/onsi/gomega: [v1.23.0 → v1.27.4](https://github.com/onsi/gomega/compare/v1.23.0...v1.27.4)
- github.com/rogpeppe/go-internal: [v1.3.0 → v1.10.0](https://github.com/rogpeppe/go-internal/compare/v1.3.0...v1.10.0)
- github.com/stretchr/objx: [v0.1.1 → v0.5.0](https://github.com/stretchr/objx/compare/v0.1.1...v0.5.0)
- github.com/stretchr/testify: [v1.8.0 → v1.8.1](https://github.com/stretchr/testify/compare/v1.8.0...v1.8.1)
- golang.org/x/mod: 86c51ed → v0.8.0
- golang.org/x/net: v0.4.0 → v0.8.0
- golang.org/x/oauth2: v0.2.0 → v0.4.0
- golang.org/x/sys: v0.3.0 → v0.6.0
- golang.org/x/term: v0.3.0 → v0.6.0
- golang.org/x/text: v0.5.0 → v0.8.0
- golang.org/x/tools: v0.1.12 → v0.7.0
- golang.org/x/xerrors: 5ec99f8 → 04be3eb
- google.golang.org/genproto: 1645502 → 76db087
- google.golang.org/grpc: v1.52.3 → v1.54.0
- k8s.io/api: v0.26.1 → v0.27.1
- k8s.io/apimachinery: v0.26.1 → v0.27.1
- k8s.io/client-go: v0.26.1 → v0.27.1
- k8s.io/csi-translation-lib: v0.26.1 → v0.27.1
- k8s.io/klog/v2: v2.90.0 → v2.90.1
- k8s.io/kube-openapi: a28e98e → 15aac26
- k8s.io/utils: 8e77b1f → a36077c
- sigs.k8s.io/json: f223a00 → bc3834c

### Removed
- github.com/PuerkitoBio/purell: [v1.1.1](https://github.com/PuerkitoBio/purell/tree/v1.1.1)
- github.com/PuerkitoBio/urlesc: [de5bf2a](https://github.com/PuerkitoBio/urlesc/tree/de5bf2a)
- github.com/elazarl/goproxy: [947c36d](https://github.com/elazarl/goproxy/tree/947c36d)
- github.com/niemeyer/pretty: [a10e7ca](https://github.com/niemeyer/pretty/tree/a10e7ca)
