# Release notes for v4.5.0

[Documentation](https://kubernetes-csi.github.io)

# Changelog since v4.4.0

## Changes by Kind

### Uncategorized

- CVE fixes:  CVE-2023-44487 ([#498](https://github.com/kubernetes-csi/external-attacher/pull/498), [@dannawang0221](https://github.com/dannawang0221))
- Update kubernetes dependencies to v1.29.0 ([#519](https://github.com/kubernetes-csi/external-attacher/pull/519), [@sunnylovestiramisu](https://github.com/sunnylovestiramisu))
- Update kubernetes dependencies to v1.29.0-rc.1 ([#516](https://github.com/kubernetes-csi/external-attacher/pull/516), [@sunnylovestiramisu](https://github.com/sunnylovestiramisu))

## Dependencies

### Added
- github.com/gorilla/websocket: [v1.5.0](https://github.com/gorilla/websocket/tree/v1.5.0)
- github.com/kubernetes-csi/csi-test/v5: [v5.2.0](https://github.com/kubernetes-csi/csi-test/v5/tree/v5.2.0)
- go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc: v0.44.0

### Changed
- cloud.google.com/go/compute: v1.19.1 → v1.23.0
- github.com/container-storage-interface/spec: [v1.8.0 → v1.9.0](https://github.com/container-storage-interface/spec/compare/v1.8.0...v1.9.0)
- github.com/emicklei/go-restful/v3: [v3.10.0 → v3.11.0](https://github.com/emicklei/go-restful/v3/compare/v3.10.0...v3.11.0)
- github.com/envoyproxy/go-control-plane: [9239064 → v0.11.1](https://github.com/envoyproxy/go-control-plane/compare/9239064...v0.11.1)
- github.com/envoyproxy/protoc-gen-validate: [v0.10.1 → v1.0.2](https://github.com/envoyproxy/protoc-gen-validate/compare/v0.10.1...v1.0.2)
- github.com/evanphx/json-patch: [v5.6.0+incompatible → v5.7.0+incompatible](https://github.com/evanphx/json-patch/compare/v5.6.0...v5.7.0)
- github.com/go-logr/logr: [v1.2.4 → v1.3.0](https://github.com/go-logr/logr/compare/v1.2.4...v1.3.0)
- github.com/golang/glog: [v1.1.0 → v1.1.2](https://github.com/golang/glog/compare/v1.1.0...v1.1.2)
- github.com/google/go-cmp: [v0.5.9 → v0.6.0](https://github.com/google/go-cmp/compare/v0.5.9...v0.6.0)
- github.com/google/uuid: [v1.3.0 → v1.4.0](https://github.com/google/uuid/compare/v1.3.0...v1.4.0)
- github.com/grpc-ecosystem/grpc-gateway/v2: [v2.7.0 → v2.16.0](https://github.com/grpc-ecosystem/grpc-gateway/v2/compare/v2.7.0...v2.16.0)
- github.com/kubernetes-csi/csi-lib-utils: [v0.14.0 → v0.17.0](https://github.com/kubernetes-csi/csi-lib-utils/compare/v0.14.0...v0.17.0)
- github.com/onsi/ginkgo/v2: [v2.9.4 → v2.13.1](https://github.com/onsi/ginkgo/v2/compare/v2.9.4...v2.13.1)
- github.com/onsi/gomega: [v1.27.6 → v1.30.0](https://github.com/onsi/gomega/compare/v1.27.6...v1.30.0)
- github.com/stretchr/testify: [v1.8.2 → v1.8.4](https://github.com/stretchr/testify/compare/v1.8.2...v1.8.4)
- github.com/yuin/goldmark: [v1.3.5 → v1.4.13](https://github.com/yuin/goldmark/compare/v1.3.5...v1.4.13)
- go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp: v0.35.1 → v0.44.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc: v1.10.0 → v1.19.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace: v1.10.0 → v1.19.0
- go.opentelemetry.io/otel/metric: v0.31.0 → v1.19.0
- go.opentelemetry.io/otel/sdk: v1.10.0 → v1.19.0
- go.opentelemetry.io/otel/trace: v1.10.0 → v1.19.0
- go.opentelemetry.io/otel: v1.10.0 → v1.19.0
- go.opentelemetry.io/proto/otlp: v0.19.0 → v1.0.0
- golang.org/x/crypto: v0.11.0 → v0.15.0
- golang.org/x/net: v0.13.0 → v0.18.0
- golang.org/x/oauth2: v0.8.0 → v0.13.0
- golang.org/x/sync: v0.2.0 → v0.4.0
- golang.org/x/sys: v0.10.0 → v0.14.0
- golang.org/x/term: v0.10.0 → v0.14.0
- golang.org/x/text: v0.11.0 → v0.14.0
- golang.org/x/tools: v0.8.0 → v0.14.0
- google.golang.org/appengine: v1.6.7 → v1.6.8
- google.golang.org/genproto/googleapis/api: dd9d682 → d307bd8
- google.golang.org/genproto/googleapis/rpc: 28d5490 → bbf56f3
- google.golang.org/genproto: 0005af6 → d783a09
- google.golang.org/grpc: v1.57.0 → v1.60.1
- google.golang.org/protobuf: v1.30.0 → v1.31.0
- k8s.io/api: v0.28.1 → v0.29.0
- k8s.io/apimachinery: v0.28.1 → v0.29.0
- k8s.io/client-go: v0.28.1 → v0.29.0
- k8s.io/component-base: v0.28.0 → v0.29.0
- k8s.io/csi-translation-lib: v0.28.1 → v0.29.0
- k8s.io/gengo: 485abfe → 9cce18d
- k8s.io/klog/v2: v2.100.1 → v2.110.1
- k8s.io/kube-openapi: 2695361 → 2dd684a
- k8s.io/utils: d93618c → 3b25d92
- sigs.k8s.io/structured-merge-diff/v4: v4.2.3 → v4.4.1

### Removed
- cloud.google.com/go: v0.34.0
- github.com/BurntSushi/toml: [v0.3.1](https://github.com/BurntSushi/toml/tree/v0.3.1)
- github.com/antihax/optional: [v1.0.0](https://github.com/antihax/optional/tree/v1.0.0)
- github.com/chzyer/logex: [v1.1.10](https://github.com/chzyer/logex/tree/v1.1.10)
- github.com/chzyer/readline: [2972be2](https://github.com/chzyer/readline/tree/2972be2)
- github.com/chzyer/test: [a1ea475](https://github.com/chzyer/test/tree/a1ea475)
- github.com/client9/misspell: [v0.3.4](https://github.com/client9/misspell/tree/v0.3.4)
- github.com/fsnotify/fsnotify: [v1.4.9](https://github.com/fsnotify/fsnotify/tree/v1.4.9)
- github.com/ghodss/yaml: [v1.0.0](https://github.com/ghodss/yaml/tree/v1.0.0)
- github.com/google/gnostic: [v0.5.7-v3refs](https://github.com/google/gnostic/tree/v0.5.7-v3refs)
- github.com/grpc-ecosystem/grpc-gateway: [v1.16.0](https://github.com/grpc-ecosystem/grpc-gateway/tree/v1.16.0)
- github.com/hpcloud/tail: [v1.0.0](https://github.com/hpcloud/tail/tree/v1.0.0)
- github.com/ianlancetaylor/demangle: [28f6c0f](https://github.com/ianlancetaylor/demangle/tree/28f6c0f)
- github.com/kubernetes-csi/csi-test/v4: [v4.4.0](https://github.com/kubernetes-csi/csi-test/v4/tree/v4.4.0)
- github.com/nxadm/tail: [v1.4.8](https://github.com/nxadm/tail/tree/v1.4.8)
- github.com/onsi/ginkgo: [v1.16.5](https://github.com/onsi/ginkgo/tree/v1.16.5)
- github.com/rogpeppe/fastuuid: [v1.2.0](https://github.com/rogpeppe/fastuuid/tree/v1.2.0)
- go.opentelemetry.io/otel/exporters/otlp/internal/retry: v1.10.0
- go.uber.org/goleak: v1.2.1
- golang.org/x/exp: 509febe
- golang.org/x/lint: d0100b6
- gopkg.in/fsnotify.v1: v1.4.7
- gopkg.in/tomb.v1: dd63297
- honnef.co/go/tools: ea95bdf
