# Release notes for v4.10.0

[Documentation](https://kubernetes-csi.github.io)

# Changelog since v4.9.0

## Changes by Kind

### API Change

- Populate VolumeError.ErrorCode field in VolumeAttachment object. ([#662](https://github.com/kubernetes-csi/external-attacher/pull/662), [@torredil](https://github.com/torredil))

### Other (Cleanup or Flake)

- Update kubernetes dependencies to v1.34.0 ([#678](https://github.com/kubernetes-csi/external-attacher/pull/678), [@dobsonj](https://github.com/dobsonj))

## Dependencies

### Added
- github.com/antihax/optional: [v1.0.0](https://github.com/antihax/optional/tree/v1.0.0)
- github.com/antlr4-go/antlr/v4: [v4.13.0](https://github.com/antlr4-go/antlr/tree/v4.13.0)
- github.com/coreos/go-oidc: [v2.3.0+incompatible](https://github.com/coreos/go-oidc/tree/v2.3.0)
- github.com/coreos/go-semver: [v0.3.1](https://github.com/coreos/go-semver/tree/v0.3.1)
- github.com/coreos/go-systemd/v22: [v22.5.0](https://github.com/coreos/go-systemd/tree/v22.5.0)
- github.com/dustin/go-humanize: [v1.0.1](https://github.com/dustin/go-humanize/tree/v1.0.1)
- github.com/fsnotify/fsnotify: [v1.9.0](https://github.com/fsnotify/fsnotify/tree/v1.9.0)
- github.com/godbus/dbus/v5: [v5.0.4](https://github.com/godbus/dbus/tree/v5.0.4)
- github.com/golang-jwt/jwt/v5: [v5.2.2](https://github.com/golang-jwt/jwt/tree/v5.2.2)
- github.com/google/cel-go: [v0.26.0](https://github.com/google/cel-go/tree/v0.26.0)
- github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus: [v1.0.1](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/providers/prometheus/v1.0.1)
- github.com/grpc-ecosystem/go-grpc-middleware/v2: [v2.3.0](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/v2.3.0)
- github.com/grpc-ecosystem/go-grpc-prometheus: [v1.2.0](https://github.com/grpc-ecosystem/go-grpc-prometheus/tree/v1.2.0)
- github.com/jonboulle/clockwork: [v0.5.0](https://github.com/jonboulle/clockwork/tree/v0.5.0)
- github.com/matttproud/golang_protobuf_extensions: [v1.0.1](https://github.com/matttproud/golang_protobuf_extensions/tree/v1.0.1)
- github.com/pquerna/cachecontrol: [v0.1.0](https://github.com/pquerna/cachecontrol/tree/v0.1.0)
- github.com/rogpeppe/fastuuid: [v1.2.0](https://github.com/rogpeppe/fastuuid/tree/v1.2.0)
- github.com/sirupsen/logrus: [v1.9.3](https://github.com/sirupsen/logrus/tree/v1.9.3)
- github.com/soheilhy/cmux: [v0.1.5](https://github.com/soheilhy/cmux/tree/v0.1.5)
- github.com/stoewer/go-strcase: [v1.3.0](https://github.com/stoewer/go-strcase/tree/v1.3.0)
- github.com/tmc/grpc-websocket-proxy: [673ab2c](https://github.com/tmc/grpc-websocket-proxy/tree/673ab2c)
- github.com/xiang90/probing: [a49e3df](https://github.com/xiang90/probing/tree/a49e3df)
- go.etcd.io/bbolt: v1.4.2
- go.etcd.io/etcd/api/v3: v3.6.4
- go.etcd.io/etcd/client/pkg/v3: v3.6.4
- go.etcd.io/etcd/client/v3: v3.6.4
- go.etcd.io/etcd/pkg/v3: v3.6.4
- go.etcd.io/etcd/server/v3: v3.6.4
- go.etcd.io/raft/v3: v3.6.0
- go.yaml.in/yaml/v2: v2.4.2
- go.yaml.in/yaml/v3: v3.0.4
- golang.org/x/exp: 8a7402a
- gopkg.in/go-jose/go-jose.v2: v2.6.3
- gopkg.in/natefinch/lumberjack.v2: v2.2.1
- k8s.io/apiserver: v0.34.0
- k8s.io/kms: v0.34.0
- sigs.k8s.io/apiserver-network-proxy/konnectivity-client: v0.33.0
- sigs.k8s.io/structured-merge-diff/v6: v6.3.0

### Changed
- cel.dev/expr: v0.20.0 → v0.24.0
- github.com/fxamacker/cbor/v2: [v2.8.0 → v2.9.0](https://github.com/fxamacker/cbor/compare/v2.8.0...v2.9.0)
- github.com/google/gnostic-models: [v0.6.9 → v0.7.0](https://github.com/google/gnostic-models/compare/v0.6.9...v0.7.0)
- github.com/grpc-ecosystem/grpc-gateway/v2: [v2.24.0 → v2.26.3](https://github.com/grpc-ecosystem/grpc-gateway/compare/v2.24.0...v2.26.3)
- github.com/modern-go/reflect2: [v1.0.2 → 35a7c28](https://github.com/modern-go/reflect2/compare/v1.0.2...35a7c28)
- go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc: v1.33.0 → v1.34.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace: v1.33.0 → v1.34.0
- go.opentelemetry.io/proto/otlp: v1.4.0 → v1.5.0
- google.golang.org/genproto/googleapis/api: 56aae31 → a0af3ef
- google.golang.org/genproto/googleapis/rpc: 56aae31 → a0af3ef
- k8s.io/api: v0.33.0 → v0.34.0
- k8s.io/apimachinery: v0.33.0 → v0.34.0
- k8s.io/client-go: v0.33.0 → v0.34.0
- k8s.io/component-base: v0.33.0 → v0.34.0
- k8s.io/csi-translation-lib: v0.33.0 → v0.34.0
- k8s.io/gengo/v2: a7b603a → 85fd79d
- k8s.io/kube-openapi: c8a335a → f3f2b99
- k8s.io/utils: 24370be → 4c0f3b2
- sigs.k8s.io/structured-merge-diff/v4: v4.7.0 → v4.6.0
- sigs.k8s.io/yaml: v1.4.0 → v1.6.0

### Removed
_Nothing has changed._
