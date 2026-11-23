# Release notes for v4.13.1

[Documentation](https://kubernetes-csi.github.io)

## Changes by Kind

### Bug or Regression

- Update golang.org/x/net to v0.62.0 and go.opentelemetry.io/otel to v1.47.0 to fix CVE-2026-78663 (CRITICAL), CVE-2026-78659, CVE-2026-78660, CVE-2026-78669, CVE-2026-97032, CVE-2026-78661 and CVE-2026-97045 and CVE-2026-128085, and golang.org/x/crypto to v0.58.0 to fix CVE-2026-78662 and CVE-2026-56855. ([#782](https://github.com/kubernetes-csi/external-attacher/pull/782), [@grant-he](https://github.com/grant-he))

### Other (Cleanup or Flake)

- Build with Go 1.26.11 to fix CVE-2026-97054. ([#781](https://github.com/kubernetes-csi/external-attacher/pull/781), [@grant-he](https://github.com/grant-he))

## Dependencies

### Added
- go.opentelemetry.io/otel/log: v1.47.0

### Changed
- go.opentelemetry.io/otel/metric: v1.46.0 → v1.47.0
- go.opentelemetry.io/otel/sdk/metric: v1.46.0 → v1.47.0
- go.opentelemetry.io/otel/sdk: v1.46.0 → v1.47.0
- go.opentelemetry.io/otel/trace: v1.46.0 → v1.47.0
- go.opentelemetry.io/otel: v1.46.0 → v1.47.0
- golang.org/x/crypto: v0.55.0 → v0.58.0
- golang.org/x/mod: v0.38.0 → v0.41.0
- golang.org/x/net: v0.58.0 → v0.62.0
- golang.org/x/sync: v0.22.0 → v0.24.0
- golang.org/x/sys: v0.47.0 → v0.49.0
- golang.org/x/term: v0.45.0 → v0.47.0
- golang.org/x/text: v0.41.0 → v0.43.0
- golang.org/x/tools: v0.48.0 → v0.51.0

### Removed
_Nothing has changed._

# Release notes for v4.13.0

[Documentation](https://kubernetes-csi.github.io)

## Changes by Kind

# Changelog since v4.12.0

## Changes by Kind

### Feature

- Updated dependencies to Kubernetes 1.37 ([#757](https://github.com/kubernetes-csi/external-attacher/pull/757), [@jsafrane](https://github.com/jsafrane))

### Bug or Regression

- Fix a bug that the controller can panic crash when it receives DeletedFinalStateUnknown deletion event. ([#305](https://github.com/kubernetes-csi/external-attacher/pull/305), [@Jiawei0227](https://github.com/Jiawei0227))

## Dependencies

### Added
- cloud.google.com/go/auth: v0.18.2
- github.com/Masterminds/semver/v3: [v3.4.0](https://github.com/Masterminds/semver/tree/v3.4.0)
- github.com/apapsch/go-jsonmerge/v2: [v2.0.0](https://github.com/apapsch/go-jsonmerge/tree/v2.0.0)
- github.com/go-openapi/analysis: [v0.25.5](https://github.com/go-openapi/analysis/tree/v0.25.5)
- github.com/go-openapi/errors: [v0.22.8](https://github.com/go-openapi/errors/tree/v0.22.8)
- github.com/go-openapi/loads: [v0.25.0](https://github.com/go-openapi/loads/tree/v0.25.0)
- github.com/go-openapi/runtime/server-middleware: [v0.30.0](https://github.com/go-openapi/runtime/tree/server-middleware/v0.30.0)
- github.com/go-openapi/runtime: [v0.33.0](https://github.com/go-openapi/runtime/tree/v0.33.0)
- github.com/go-openapi/spec: [v0.22.9](https://github.com/go-openapi/spec/tree/v0.22.9)
- github.com/go-openapi/strfmt: [v0.27.0](https://github.com/go-openapi/strfmt/tree/v0.27.0)
- github.com/go-openapi/swag/pools: [v0.29.1](https://github.com/go-openapi/swag/tree/pools/v0.29.1)
- github.com/go-openapi/validate: [v0.26.1](https://github.com/go-openapi/validate/tree/v0.26.1)
- github.com/go-viper/mapstructure/v2: [v2.5.0](https://github.com/go-viper/mapstructure/tree/v2.5.0)
- github.com/google/s2a-go: [v0.1.9](https://github.com/google/s2a-go/tree/v0.1.9)
- github.com/googleapis/enterprise-certificate-proxy: [v0.3.11](https://github.com/googleapis/enterprise-certificate-proxy/tree/v0.3.11)
- github.com/googleapis/gax-go/v2: [v2.17.0](https://github.com/googleapis/gax-go/tree/v2.17.0)
- github.com/oapi-codegen/runtime: [v1.6.0](https://github.com/oapi-codegen/runtime/tree/v1.6.0)
- github.com/oklog/ulid/v2: [v2.1.1](https://github.com/oklog/ulid/tree/v2.1.1)
- go.opentelemetry.io/otel/exporters/stdout/stdouttrace: v1.45.0
- go.uber.org/mock: v0.6.0

### Changed
- buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go: 8976f5b → 52f3232
- buf.build/go/protovalidate: v0.12.0 → v1.0.0
- cel.dev/expr: v0.25.2 → v0.25.3
- github.com/Azure/go-ansiterm: [306776e → faa5f7b](https://github.com/Azure/go-ansiterm/compare/306776e...faa5f7b)
- github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp: [v1.31.0 → v1.33.0](https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/compare/detectors/gcp/v1.31.0...detectors/gcp/v1.33.0)
- github.com/container-storage-interface/spec: [v1.12.0 → v1.13.0](https://github.com/container-storage-interface/spec/compare/v1.12.0...v1.13.0)
- github.com/felixge/httpsnoop: [v1.0.4 → v1.1.0](https://github.com/felixge/httpsnoop/compare/v1.0.4...v1.1.0)
- github.com/fxamacker/cbor/v2: [v2.9.2 → v2.9.3](https://github.com/fxamacker/cbor/compare/v2.9.2...v2.9.3)
- github.com/go-logr/logr: [v1.4.3 → v1.4.4](https://github.com/go-logr/logr/compare/v1.4.3...v1.4.4)
- github.com/go-openapi/jsonpointer: [v0.23.1 → v1.0.0](https://github.com/go-openapi/jsonpointer/compare/v0.23.1...v1.0.0)
- github.com/go-openapi/jsonreference: [v0.21.5 → v1.0.1](https://github.com/go-openapi/jsonreference/compare/v0.21.5...v1.0.1)
- github.com/go-openapi/swag/cmdutils: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/cmdutils/v0.26.0...cmdutils/v0.29.1)
- github.com/go-openapi/swag/conv: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/conv/v0.26.0...conv/v0.29.1)
- github.com/go-openapi/swag/fileutils: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/fileutils/v0.26.0...fileutils/v0.29.1)
- github.com/go-openapi/swag/jsonutils/fixtures_test: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/jsonutils/fixtures_test/v0.26.0...jsonutils/fixtures_test/v0.29.1)
- github.com/go-openapi/swag/jsonutils: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/jsonutils/v0.26.0...jsonutils/v0.29.1)
- github.com/go-openapi/swag/loading: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/loading/v0.26.0...loading/v0.29.1)
- github.com/go-openapi/swag/mangling: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/mangling/v0.26.0...mangling/v0.29.1)
- github.com/go-openapi/swag/netutils: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/netutils/v0.26.0...netutils/v0.29.1)
- github.com/go-openapi/swag/stringutils: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/stringutils/v0.26.0...stringutils/v0.29.1)
- github.com/go-openapi/swag/typeutils: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/typeutils/v0.26.0...typeutils/v0.29.1)
- github.com/go-openapi/swag/yamlutils: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/yamlutils/v0.26.0...yamlutils/v0.29.1)
- github.com/go-openapi/swag: [v0.26.0 → v0.29.1](https://github.com/go-openapi/swag/compare/v0.26.0...v0.29.1)
- github.com/go-openapi/testify/enable/yaml/v2: [v2.4.2 → v2.6.1](https://github.com/go-openapi/testify/compare/enable/yaml/v2/v2.4.2...enable/yaml/v2/v2.6.1)
- github.com/go-openapi/testify/v2: [v2.4.2 → v2.6.1](https://github.com/go-openapi/testify/compare/v2.4.2...v2.6.1)
- github.com/golang-jwt/jwt/v5: [v5.3.0 → v5.3.1](https://github.com/golang-jwt/jwt/compare/v5.3.0...v5.3.1)
- github.com/google/cel-go: [v0.28.1 → v0.30.0](https://github.com/google/cel-go/compare/v0.28.1...v0.30.0)
- github.com/google/pprof: [40e02aa → 545e8a4](https://github.com/google/pprof/compare/40e02aa...545e8a4)
- github.com/grpc-ecosystem/go-grpc-middleware/v2: [v2.3.3 → v2.3.4](https://github.com/grpc-ecosystem/go-grpc-middleware/compare/v2.3.3...v2.3.4)
- github.com/grpc-ecosystem/grpc-gateway/v2: [v2.29.0 → v2.30.0](https://github.com/grpc-ecosystem/grpc-gateway/compare/v2.29.0...v2.30.0)
- github.com/klauspost/compress: [v1.18.0 → v1.19.1](https://github.com/klauspost/compress/compare/v1.18.0...v1.19.1)
- github.com/kubernetes-csi/csi-test/v5: [v5.4.0 → v5.6.0](https://github.com/kubernetes-csi/csi-test/compare/v5.4.0...v5.6.0)
- github.com/moby/term: [v0.5.0 → v0.5.2](https://github.com/moby/term/compare/v0.5.0...v0.5.2)
- github.com/onsi/ginkgo/v2: [v2.22.0 → v2.32.1](https://github.com/onsi/ginkgo/compare/v2.22.0...v2.32.1)
- github.com/onsi/gomega: [v1.36.1 → v1.42.1](https://github.com/onsi/gomega/compare/v1.36.1...v1.42.1)
- github.com/prometheus/client_golang: [v1.23.2 → v1.24.1](https://github.com/prometheus/client_golang/compare/v1.23.2...v1.24.1)
- github.com/prometheus/common: [v0.67.5 → v0.70.1](https://github.com/prometheus/common/compare/v0.67.5...v0.70.1)
- github.com/prometheus/procfs: [v0.20.1 → v0.22.0](https://github.com/prometheus/procfs/compare/v0.20.1...v0.22.0)
- github.com/sirupsen/logrus: [v1.9.3 → v1.9.4](https://github.com/sirupsen/logrus/compare/v1.9.3...v1.9.4)
- github.com/spiffe/go-spiffe/v2: [v2.6.0 → v2.7.0](https://github.com/spiffe/go-spiffe/compare/v2.6.0...v2.7.0)
- github.com/stoewer/go-strcase: [v1.3.0 → v1.3.1](https://github.com/stoewer/go-strcase/compare/v1.3.0...v1.3.1)
- github.com/stretchr/objx: [v0.5.2 → v0.5.3](https://github.com/stretchr/objx/compare/v0.5.2...v0.5.3)
- github.com/stretchr/testify: [v1.11.1 → v1.12.1](https://github.com/stretchr/testify/compare/v1.11.1...v1.12.1)
- github.com/yuin/goldmark: [v1.3.5 → v1.4.13](https://github.com/yuin/goldmark/compare/v1.3.5...v1.4.13)
- go.etcd.io/bbolt: v1.4.3 → v1.5.0
- go.etcd.io/etcd/api/v3: v3.6.11 → v3.7.1
- go.etcd.io/etcd/client/pkg/v3: v3.6.11 → v3.7.1
- go.etcd.io/etcd/client/v3: v3.6.11 → v3.7.1
- go.etcd.io/etcd/pkg/v3: v3.6.8 → v3.7.0
- go.etcd.io/etcd/server/v3: v3.6.8 → v3.7.0
- go.etcd.io/raft/v3: v3.6.0 → v3.7.0
- go.opentelemetry.io/contrib/detectors/gcp: v1.42.0 → v1.44.0
- go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc: v0.68.0 → v0.71.0
- go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp: v0.68.0 → v0.71.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc: v1.43.0 → v1.46.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace: v1.43.0 → v1.46.0
- go.opentelemetry.io/otel/metric: v1.43.0 → v1.46.0
- go.opentelemetry.io/otel/sdk/metric: v1.43.0 → v1.46.0
- go.opentelemetry.io/otel/sdk: v1.43.0 → v1.46.0
- go.opentelemetry.io/otel/trace: v1.43.0 → v1.46.0
- go.opentelemetry.io/otel: v1.43.0 → v1.46.0
- go.opentelemetry.io/proto/otlp: v1.10.0 → v1.11.0
- go.yaml.in/yaml/v3: v3.0.4 → v3.0.5
- golang.org/x/crypto: v0.51.0 → v0.55.0
- golang.org/x/exp: 944ab1f → 746e56f
- golang.org/x/mod: v0.35.0 → v0.38.0
- golang.org/x/net: v0.54.0 → v0.58.0
- golang.org/x/sync: v0.20.0 → v0.22.0
- golang.org/x/sys: v0.44.0 → v0.47.0
- golang.org/x/term: v0.43.0 → v0.45.0
- golang.org/x/text: v0.37.0 → v0.41.0
- golang.org/x/tools: v0.44.0 → v0.48.0
- google.golang.org/genproto/googleapis/api: afd174a → 08b0e42
- google.golang.org/genproto/googleapis/rpc: afd174a → da73d73
- google.golang.org/grpc: v1.81.1 → v1.83.2
- google.golang.org/protobuf: f2248ac → v1.36.12
- k8s.io/api: v0.36.1 → v0.37.0
- k8s.io/apimachinery: v0.36.1 → v0.37.0
- k8s.io/apiserver: v0.36.1 → v0.37.0
- k8s.io/client-go: v0.36.1 → v0.37.0
- k8s.io/component-base: v0.36.1 → v0.37.0
- k8s.io/csi-translation-lib: v0.36.1 → v0.37.0
- k8s.io/gengo/v2: 85fd79d → ec3ebc5
- k8s.io/kms: v0.36.1 → v0.37.0
- k8s.io/kube-openapi: 43fb72c → d427ff9
- k8s.io/streaming: v0.36.1 → v0.37.0
- k8s.io/utils: b8788ab → be93311
- sigs.k8s.io/apiserver-network-proxy/konnectivity-client: v0.34.0 → v0.36.0
- sigs.k8s.io/structured-merge-diff/v6: v6.4.0 → v6.4.2

### Removed
- github.com/antihax/optional: [v1.0.0](https://github.com/antihax/optional/tree/v1.0.0)
- github.com/go-openapi/swag/jsonname: [v0.26.0](https://github.com/go-openapi/swag/tree/jsonname/v0.26.0)
- github.com/gogo/protobuf: [v1.3.2](https://github.com/gogo/protobuf/tree/v1.3.2)
- github.com/golang/mock: [v1.6.0](https://github.com/golang/mock/tree/v1.6.0)
- github.com/kisielk/errcheck: [v1.5.0](https://github.com/kisielk/errcheck/tree/v1.5.0)
- github.com/kisielk/gotool: [v1.0.0](https://github.com/kisielk/gotool/tree/v1.0.0)
- golang.org/x/xerrors: 5ec99f8
