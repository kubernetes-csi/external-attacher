# Release notes for v4.9.0

[Documentation](https://kubernetes-csi.github.io)

# Changelog since v4.8.0

## Changes by Kind

### Feature

- Add a new `--automaxprocs` flag to set the `GOMAXPROCS` environment variable to match the configured Linux container CPU quota. ([#650](https://github.com/kubernetes-csi/external-attacher/pull/650), [@nixpanic](https://github.com/nixpanic))
- Serve additional leader election, work queue, process, and Go runtime metrics ([#630](https://github.com/kubernetes-csi/external-attacher/pull/630), [@AndrewSirenko](https://github.com/AndrewSirenko))

### Bug or Regression

- Fixed repeated call to ControllerUnpublishVolume by reordering its finalizer removal and marking VolumeAttachment as detached. The VolumeAttachment can now be deleted in the API server before it's marked as `attached: false`. ([#624](https://github.com/kubernetes-csi/external-attacher/pull/624), [@jsafrane](https://github.com/jsafrane))

### Uncategorized

- Update kubernetes dependencies to v1.33.0 ([#647](https://github.com/kubernetes-csi/external-attacher/pull/647), [@Aishwarya-Hebbar](https://github.com/Aishwarya-Hebbar))

## Dependencies

### Added
- github.com/envoyproxy/go-control-plane/envoy: [v1.32.4](https://github.com/envoyproxy/go-control-plane/tree/envoy/v1.32.4)
- github.com/envoyproxy/go-control-plane/ratelimit: [v0.1.0](https://github.com/envoyproxy/go-control-plane/tree/ratelimit/v0.1.0)
- github.com/go-jose/go-jose/v4: [v4.0.4](https://github.com/go-jose/go-jose/tree/v4.0.4)
- github.com/prashantv/gostub: [v1.1.0](https://github.com/prashantv/gostub/tree/v1.1.0)
- github.com/spiffe/go-spiffe/v2: [v2.5.0](https://github.com/spiffe/go-spiffe/tree/v2.5.0)
- github.com/zeebo/errs: [v1.4.0](https://github.com/zeebo/errs/tree/v1.4.0)
- go.uber.org/automaxprocs: v1.6.0
- sigs.k8s.io/randfill: v1.0.0

### Changed
- cel.dev/expr: v0.16.2 → v0.20.0
- cloud.google.com/go/compute/metadata: v0.5.2 → v0.6.0
- github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp: [v1.24.2 → v1.26.0](https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/compare/detectors/gcp/v1.24.2...detectors/gcp/v1.26.0)
- github.com/cncf/xds/go: [b4127c9 → 2f00578](https://github.com/cncf/xds/compare/b4127c9...2f00578)
- github.com/cpuguy83/go-md2man/v2: [v2.0.4 → v2.0.6](https://github.com/cpuguy83/go-md2man/compare/v2.0.4...v2.0.6)
- github.com/emicklei/go-restful/v3: [v3.12.1 → v3.12.2](https://github.com/emicklei/go-restful/compare/v3.12.1...v3.12.2)
- github.com/envoyproxy/go-control-plane: [v0.13.1 → v0.13.4](https://github.com/envoyproxy/go-control-plane/compare/v0.13.1...v0.13.4)
- github.com/envoyproxy/protoc-gen-validate: [v1.1.0 → v1.2.1](https://github.com/envoyproxy/protoc-gen-validate/compare/v1.1.0...v1.2.1)
- github.com/evanphx/json-patch: [v5.9.0+incompatible → v5.9.11+incompatible](https://github.com/evanphx/json-patch/compare/v5.9.0...v5.9.11)
- github.com/fxamacker/cbor/v2: [v2.7.0 → v2.8.0](https://github.com/fxamacker/cbor/compare/v2.7.0...v2.8.0)
- github.com/go-openapi/jsonpointer: [v0.21.0 → v0.21.1](https://github.com/go-openapi/jsonpointer/compare/v0.21.0...v0.21.1)
- github.com/go-openapi/swag: [v0.23.0 → v0.23.1](https://github.com/go-openapi/swag/compare/v0.23.0...v0.23.1)
- github.com/golang/glog: [v1.2.2 → v1.2.4](https://github.com/golang/glog/compare/v1.2.2...v1.2.4)
- github.com/google/btree: [v1.0.1 → v1.1.3](https://github.com/google/btree/compare/v1.0.1...v1.1.3)
- github.com/google/go-cmp: [v0.6.0 → v0.7.0](https://github.com/google/go-cmp/compare/v0.6.0...v0.7.0)
- github.com/google/gofuzz: [v1.2.0 → v1.0.0](https://github.com/google/gofuzz/compare/v1.2.0...v1.0.0)
- github.com/gorilla/websocket: [v1.5.0 → e064f32](https://github.com/gorilla/websocket/compare/v1.5.0...e064f32)
- github.com/grpc-ecosystem/grpc-gateway/v2: [v2.20.0 → v2.24.0](https://github.com/grpc-ecosystem/grpc-gateway/compare/v2.20.0...v2.24.0)
- github.com/klauspost/compress: [v1.17.11 → v1.18.0](https://github.com/klauspost/compress/compare/v1.17.11...v1.18.0)
- github.com/kubernetes-csi/csi-lib-utils: [v0.20.0 → v0.22.0](https://github.com/kubernetes-csi/csi-lib-utils/compare/v0.20.0...v0.22.0)
- github.com/prometheus/client_golang: [v1.20.5 → v1.22.0](https://github.com/prometheus/client_golang/compare/v1.20.5...v1.22.0)
- github.com/prometheus/client_model: [v0.6.1 → v0.6.2](https://github.com/prometheus/client_model/compare/v0.6.1...v0.6.2)
- github.com/prometheus/common: [v0.61.0 → v0.64.0](https://github.com/prometheus/common/compare/v0.61.0...v0.64.0)
- github.com/prometheus/procfs: [v0.15.1 → v0.16.1](https://github.com/prometheus/procfs/compare/v0.15.1...v0.16.1)
- github.com/spf13/cobra: [v1.8.1 → v1.9.1](https://github.com/spf13/cobra/compare/v1.8.1...v1.9.1)
- github.com/spf13/pflag: [v1.0.5 → v1.0.6](https://github.com/spf13/pflag/compare/v1.0.5...v1.0.6)
- github.com/stretchr/objx: [v0.1.0 → v0.5.2](https://github.com/stretchr/objx/compare/v0.1.0...v0.5.2)
- go.opentelemetry.io/contrib/detectors/gcp: v1.31.0 → v1.34.0
- go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc: v0.58.0 → v0.60.0
- go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp: v0.53.0 → v0.58.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc: v1.27.0 → v1.33.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace: v1.28.0 → v1.33.0
- go.opentelemetry.io/otel/metric: v1.33.0 → v1.35.0
- go.opentelemetry.io/otel/sdk/metric: v1.31.0 → v1.34.0
- go.opentelemetry.io/otel/sdk: v1.31.0 → v1.34.0
- go.opentelemetry.io/otel/trace: v1.33.0 → v1.35.0
- go.opentelemetry.io/otel: v1.33.0 → v1.35.0
- go.opentelemetry.io/proto/otlp: v1.3.1 → v1.4.0
- golang.org/x/crypto: v0.30.0 → v0.38.0
- golang.org/x/net: v0.32.0 → v0.40.0
- golang.org/x/oauth2: v0.24.0 → v0.30.0
- golang.org/x/sync: v0.10.0 → v0.14.0
- golang.org/x/sys: v0.28.0 → v0.33.0
- golang.org/x/term: v0.27.0 → v0.32.0
- golang.org/x/text: v0.21.0 → v0.25.0
- golang.org/x/time: v0.8.0 → v0.11.0
- google.golang.org/genproto/googleapis/api: 796eee8 → 56aae31
- google.golang.org/genproto/googleapis/rpc: 9240e9c → 56aae31
- google.golang.org/grpc: v1.69.0 → v1.72.1
- google.golang.org/protobuf: v1.36.0 → v1.36.6
- k8s.io/api: v0.32.0 → v0.33.0
- k8s.io/apimachinery: v0.32.0 → v0.33.0
- k8s.io/client-go: v0.32.0 → v0.33.0
- k8s.io/component-base: v0.32.0 → v0.33.0
- k8s.io/csi-translation-lib: v0.32.0 → v0.33.0
- k8s.io/kube-openapi: 2c72e55 → c8a335a
- sigs.k8s.io/structured-merge-diff/v4: v4.5.0 → v4.7.0

### Removed
- github.com/census-instrumentation/opencensus-proto: [v0.4.1](https://github.com/census-instrumentation/opencensus-proto/tree/v0.4.1)
- github.com/go-kit/log: [v0.2.1](https://github.com/go-kit/log/tree/v0.2.1)
- github.com/go-logfmt/logfmt: [v0.5.1](https://github.com/go-logfmt/logfmt/tree/v0.5.1)
