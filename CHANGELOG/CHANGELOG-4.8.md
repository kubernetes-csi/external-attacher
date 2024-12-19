# Release notes for v4.8.0

[Documentation](https://kubernetes-csi.github.io)

# Changelog since v4.7.0

## Changes by Kind

### Bug or Regression

- Changing distroless image back to multiarch ([#267](https://github.com/kubernetes-csi/external-attacher/pull/267), [@namrata-ibm](https://github.com/namrata-ibm))

### Other (Cleanup or Flake)

- Update Kubernetes dependencies to 1.32.0 ([#613](https://github.com/kubernetes-csi/external-attacher/pull/613), [@dfajmon](https://github.com/dfajmon))

## Dependencies

### Added
- github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp: [v1.24.2](https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/tree/detectors/gcp/v1.24.2)
- github.com/planetscale/vtprotobuf: [0393e58](https://github.com/planetscale/vtprotobuf/tree/0393e58)
- go.opentelemetry.io/auto/sdk: v1.1.0
- go.opentelemetry.io/contrib/detectors/gcp: v1.31.0
- go.opentelemetry.io/otel/sdk/metric: v1.31.0

### Changed
- cel.dev/expr: v0.15.0 → v0.16.2
- cloud.google.com/go/compute/metadata: v0.3.0 → v0.5.2
- github.com/Azure/go-ansiterm: [d185dfc → 306776e](https://github.com/Azure/go-ansiterm/compare/d185dfc...306776e)
- github.com/NYTimes/gziphandler: [56545f4 → v1.1.1](https://github.com/NYTimes/gziphandler/compare/56545f4...v1.1.1)
- github.com/cncf/xds/go: [555b57e → b4127c9](https://github.com/cncf/xds/compare/555b57e...b4127c9)
- github.com/container-storage-interface/spec: [v1.10.0 → v1.11.0](https://github.com/container-storage-interface/spec/compare/v1.10.0...v1.11.0)
- github.com/envoyproxy/go-control-plane: [v0.12.0 → v0.13.1](https://github.com/envoyproxy/go-control-plane/compare/v0.12.0...v0.13.1)
- github.com/envoyproxy/protoc-gen-validate: [v1.0.4 → v1.1.0](https://github.com/envoyproxy/protoc-gen-validate/compare/v1.0.4...v1.1.0)
- github.com/golang/glog: [v1.2.1 → v1.2.2](https://github.com/golang/glog/compare/v1.2.1...v1.2.2)
- github.com/google/gnostic-models: [v0.6.8 → v0.6.9](https://github.com/google/gnostic-models/compare/v0.6.8...v0.6.9)
- github.com/google/pprof: [4bfdf5a → d1b30fe](https://github.com/google/pprof/compare/4bfdf5a...d1b30fe)
- github.com/gregjones/httpcache: [9cad4c3 → 901d907](https://github.com/gregjones/httpcache/compare/9cad4c3...901d907)
- github.com/klauspost/compress: [v1.17.9 → v1.17.11](https://github.com/klauspost/compress/compare/v1.17.9...v1.17.11)
- github.com/kubernetes-csi/csi-lib-utils: [v0.19.0 → v0.20.0](https://github.com/kubernetes-csi/csi-lib-utils/compare/v0.19.0...v0.20.0)
- github.com/kubernetes-csi/csi-test/v5: [v5.3.0 → v5.3.1](https://github.com/kubernetes-csi/csi-test/compare/v5.3.0...v5.3.1)
- github.com/mailru/easyjson: [v0.7.7 → v0.9.0](https://github.com/mailru/easyjson/compare/v0.7.7...v0.9.0)
- github.com/moby/spdystream: [v0.4.0 → v0.5.0](https://github.com/moby/spdystream/compare/v0.4.0...v0.5.0)
- github.com/onsi/ginkgo/v2: [v2.19.0 → v2.21.0](https://github.com/onsi/ginkgo/compare/v2.19.0...v2.21.0)
- github.com/onsi/gomega: [v1.33.1 → v1.35.1](https://github.com/onsi/gomega/compare/v1.33.1...v1.35.1)
- github.com/prometheus/client_golang: [v1.20.0 → v1.20.5](https://github.com/prometheus/client_golang/compare/v1.20.0...v1.20.5)
- github.com/prometheus/common: [v0.55.0 → v0.61.0](https://github.com/prometheus/common/compare/v0.55.0...v0.61.0)
- github.com/rogpeppe/go-internal: [v1.12.0 → v1.13.1](https://github.com/rogpeppe/go-internal/compare/v1.12.0...v1.13.1)
- github.com/stretchr/testify: [v1.9.0 → v1.10.0](https://github.com/stretchr/testify/compare/v1.9.0...v1.10.0)
- go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc: v0.53.0 → v0.58.0
- go.opentelemetry.io/otel/metric: v1.28.0 → v1.33.0
- go.opentelemetry.io/otel/sdk: v1.28.0 → v1.31.0
- go.opentelemetry.io/otel/trace: v1.28.0 → v1.33.0
- go.opentelemetry.io/otel: v1.28.0 → v1.33.0
- golang.org/x/crypto: v0.26.0 → v0.30.0
- golang.org/x/mod: v0.17.0 → v0.20.0
- golang.org/x/net: v0.28.0 → v0.32.0
- golang.org/x/oauth2: v0.22.0 → v0.24.0
- golang.org/x/sync: v0.8.0 → v0.10.0
- golang.org/x/sys: v0.24.0 → v0.28.0
- golang.org/x/term: v0.23.0 → v0.27.0
- golang.org/x/text: v0.17.0 → v0.21.0
- golang.org/x/time: v0.6.0 → v0.8.0
- golang.org/x/tools: e35e4cc → v0.26.0
- golang.org/x/xerrors: 04be3eb → 5ec99f8
- google.golang.org/genproto/googleapis/api: 5315273 → 796eee8
- google.golang.org/genproto/googleapis/rpc: f6361c8 → 9240e9c
- google.golang.org/grpc: v1.65.0 → v1.69.0
- google.golang.org/protobuf: v1.34.2 → v1.36.0
- k8s.io/api: v0.31.0 → v0.32.0
- k8s.io/apimachinery: v0.31.0 → v0.32.0
- k8s.io/client-go: v0.31.0 → v0.32.0
- k8s.io/component-base: v0.31.0 → v0.32.0
- k8s.io/csi-translation-lib: v0.31.0 → v0.32.0
- k8s.io/gengo/v2: 51d4e06 → a7b603a
- k8s.io/kube-openapi: 70dd376 → 2c72e55
- k8s.io/utils: 18e509b → 24370be
- sigs.k8s.io/json: bc3834c → cfa47c3
- sigs.k8s.io/structured-merge-diff/v4: v4.4.1 → v4.5.0

### Removed
- github.com/asaskevich/govalidator: [f61b66f](https://github.com/asaskevich/govalidator/tree/f61b66f)
- github.com/golang/groupcache: [41bb18b](https://github.com/golang/groupcache/tree/41bb18b)
- github.com/imdario/mergo: [v0.3.13](https://github.com/imdario/mergo/tree/v0.3.13)
