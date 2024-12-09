# Release notes for v3.3.0

[Documentation](https://kubernetes-csi.github.io)
# Changelog since v3.2.0

## Changes by Kind

### Feature

- Add command line arguments to configure leader election options ([#313](https://github.com/kubernetes-csi/external-attacher/pull/313), [@RaunakShah](https://github.com/RaunakShah))
- Adds mappings for PV access modes to new CSI access modes: `SINGLE_NODE_SINGLE_WRITER` and `SINGLE_NODE_MULTI_WRITER`. ([#308](https://github.com/kubernetes-csi/external-attacher/pull/308), [@chrishenzie](https://github.com/chrishenzie))
- Updates Kubernetes dependencies to v1.22.0 ([#321](https://github.com/kubernetes-csi/external-attacher/pull/321), [@chrishenzie](https://github.com/chrishenzie)) [SIG Storage]

### Bug or Regression

- Fix a bug that the controller can panic crash when it receives DeletedFinalStateUnknown deletion event. ([#304](https://github.com/kubernetes-csi/external-attacher/pull/304), [@Jiawei0227](https://github.com/Jiawei0227))

### Other (Cleanup or Flake)

- Updates container-storage-interface dependency to v1.5.0 ([#312](https://github.com/kubernetes-csi/external-attacher/pull/312), [@chrishenzie](https://github.com/chrishenzie))
- Reuse the same gRPC CSI client for all CSI driver calls ([#318](https://github.com/kubernetes-csi/external-attacher/pull/318), [@yeya24](https://github.com/yeya24))

## Dependencies

### Added
- cloud.google.com/go/firestore: v1.1.0
- github.com/OneOfOne/xxhash: [v1.2.2](https://github.com/OneOfOne/xxhash/tree/v1.2.2)
- github.com/antihax/optional: [v1.0.0](https://github.com/antihax/optional/tree/v1.0.0)
- github.com/benbjohnson/clock: [v1.0.3](https://github.com/benbjohnson/clock/tree/v1.0.3)
- github.com/bketelsen/crypt: [5cbc8cc](https://github.com/bketelsen/crypt/tree/5cbc8cc)
- github.com/cespare/xxhash: [v1.1.0](https://github.com/cespare/xxhash/tree/v1.1.0)
- github.com/coreos/bbolt: [v1.3.2](https://github.com/coreos/bbolt/tree/v1.3.2)
- github.com/coreos/etcd: [v3.3.13+incompatible](https://github.com/coreos/etcd/tree/v3.3.13)
- github.com/dgryski/go-sip13: [e10d5fe](https://github.com/dgryski/go-sip13/tree/e10d5fe)
- github.com/felixge/httpsnoop: [v1.0.1](https://github.com/felixge/httpsnoop/tree/v1.0.1)
- github.com/go-kit/log: [v0.1.0](https://github.com/go-kit/log/tree/v0.1.0)
- github.com/hashicorp/hcl: [v1.0.0](https://github.com/hashicorp/hcl/tree/v1.0.0)
- github.com/magiconair/properties: [v1.8.1](https://github.com/magiconair/properties/tree/v1.8.1)
- github.com/moby/spdystream: [v0.2.0](https://github.com/moby/spdystream/tree/v0.2.0)
- github.com/niemeyer/pretty: [a10e7ca](https://github.com/niemeyer/pretty/tree/a10e7ca)
- github.com/nxadm/tail: [v1.4.4](https://github.com/nxadm/tail/tree/v1.4.4)
- github.com/oklog/ulid: [v1.3.1](https://github.com/oklog/ulid/tree/v1.3.1)
- github.com/pelletier/go-toml: [v1.2.0](https://github.com/pelletier/go-toml/tree/v1.2.0)
- github.com/prometheus/tsdb: [v0.7.1](https://github.com/prometheus/tsdb/tree/v0.7.1)
- github.com/spaolacci/murmur3: [f09979e](https://github.com/spaolacci/murmur3/tree/f09979e)
- github.com/spf13/cast: [v1.3.0](https://github.com/spf13/cast/tree/v1.3.0)
- github.com/spf13/jwalterweatherman: [v1.0.0](https://github.com/spf13/jwalterweatherman/tree/v1.0.0)
- github.com/spf13/viper: [v1.7.0](https://github.com/spf13/viper/tree/v1.7.0)
- github.com/subosito/gotenv: [v1.2.0](https://github.com/subosito/gotenv/tree/v1.2.0)
- go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp: v0.20.0
- go.opentelemetry.io/contrib: v0.20.0
- go.opentelemetry.io/otel/exporters/otlp: v0.20.0
- go.opentelemetry.io/otel/metric: v0.20.0
- go.opentelemetry.io/otel/oteltest: v0.20.0
- go.opentelemetry.io/otel/sdk/export/metric: v0.20.0
- go.opentelemetry.io/otel/sdk/metric: v0.20.0
- go.opentelemetry.io/otel/sdk: v0.20.0
- go.opentelemetry.io/otel/trace: v0.20.0
- go.opentelemetry.io/otel: v0.20.0
- go.opentelemetry.io/proto/otlp: v0.7.0
- gopkg.in/ini.v1: v1.51.0

### Changed
- github.com/Azure/go-ansiterm: [d6e3b33 → d185dfc](https://github.com/Azure/go-ansiterm/compare/d6e3b33...d185dfc)
- github.com/Azure/go-autorest/autorest/adal: [v0.9.5 → v0.9.13](https://github.com/Azure/go-autorest/autorest/adal/compare/v0.9.5...v0.9.13)
- github.com/Azure/go-autorest/autorest: [v0.11.1 → v0.11.18](https://github.com/Azure/go-autorest/autorest/compare/v0.11.1...v0.11.18)
- github.com/Azure/go-autorest/logger: [v0.2.0 → v0.2.1](https://github.com/Azure/go-autorest/logger/compare/v0.2.0...v0.2.1)
- github.com/cncf/udpa/go: [efcf912 → 5459f2c](https://github.com/cncf/udpa/go/compare/efcf912...5459f2c)
- github.com/container-storage-interface/spec: [v1.3.0 → v1.5.0](https://github.com/container-storage-interface/spec/compare/v1.3.0...v1.5.0)
- github.com/coreos/go-semver: [v0.2.0 → v0.3.0](https://github.com/coreos/go-semver/compare/v0.2.0...v0.3.0)
- github.com/coreos/go-systemd: [39ca1b0 → 95778df](https://github.com/coreos/go-systemd/compare/39ca1b0...95778df)
- github.com/coreos/pkg: [3ac0863 → 399ea9e](https://github.com/coreos/pkg/compare/3ac0863...399ea9e)
- github.com/cpuguy83/go-md2man/v2: [f79a8a8 → v2.0.0](https://github.com/cpuguy83/go-md2man/v2/compare/f79a8a8...v2.0.0)
- github.com/creack/pty: [v1.1.7 → v1.1.11](https://github.com/creack/pty/compare/v1.1.7...v1.1.11)
- github.com/envoyproxy/go-control-plane: [v0.9.7 → 668b12f](https://github.com/envoyproxy/go-control-plane/compare/v0.9.7...668b12f)
- github.com/evanphx/json-patch: [v4.9.0+incompatible → v4.11.0+incompatible](https://github.com/evanphx/json-patch/compare/v4.9.0...v4.11.0)
- github.com/form3tech-oss/jwt-go: [v3.2.2+incompatible → v3.2.3+incompatible](https://github.com/form3tech-oss/jwt-go/compare/v3.2.2...v3.2.3)
- github.com/go-kit/kit: [v0.10.0 → v0.9.0](https://github.com/go-kit/kit/compare/v0.10.0...v0.9.0)
- github.com/gogo/protobuf: [v1.3.1 → v1.3.2](https://github.com/gogo/protobuf/compare/v1.3.1...v1.3.2)
- github.com/golang/groupcache: [8c9f03a → 41bb18b](https://github.com/golang/groupcache/compare/8c9f03a...41bb18b)
- github.com/golang/protobuf: [v1.4.3 → v1.5.2](https://github.com/golang/protobuf/compare/v1.4.3...v1.5.2)
- github.com/google/btree: [v1.0.0 → v1.0.1](https://github.com/google/btree/compare/v1.0.0...v1.0.1)
- github.com/google/go-cmp: [v0.5.4 → v0.5.5](https://github.com/google/go-cmp/compare/v0.5.4...v0.5.5)
- github.com/googleapis/gnostic: [v0.5.3 → v0.5.5](https://github.com/googleapis/gnostic/compare/v0.5.3...v0.5.5)
- github.com/gorilla/websocket: [4201258 → v1.4.2](https://github.com/gorilla/websocket/compare/4201258...v1.4.2)
- github.com/grpc-ecosystem/go-grpc-middleware: [f849b54 → v1.0.0](https://github.com/grpc-ecosystem/go-grpc-middleware/compare/f849b54...v1.0.0)
- github.com/grpc-ecosystem/grpc-gateway: [v1.9.5 → v1.16.0](https://github.com/grpc-ecosystem/grpc-gateway/compare/v1.9.5...v1.16.0)
- github.com/hashicorp/consul/api: [v1.3.0 → v1.1.0](https://github.com/hashicorp/consul/api/compare/v1.3.0...v1.1.0)
- github.com/hashicorp/consul/sdk: [v0.3.0 → v0.1.1](https://github.com/hashicorp/consul/sdk/compare/v0.3.0...v0.1.1)
- github.com/hashicorp/golang-lru: [v0.5.4 → v0.5.1](https://github.com/hashicorp/golang-lru/compare/v0.5.4...v0.5.1)
- github.com/imdario/mergo: [v0.3.11 → v0.3.12](https://github.com/imdario/mergo/compare/v0.3.11...v0.3.12)
- github.com/json-iterator/go: [v1.1.10 → v1.1.11](https://github.com/json-iterator/go/compare/v1.1.10...v1.1.11)
- github.com/kisielk/errcheck: [v1.2.0 → v1.5.0](https://github.com/kisielk/errcheck/compare/v1.2.0...v1.5.0)
- github.com/kr/pty: [v1.1.5 → v1.1.1](https://github.com/kr/pty/compare/v1.1.5...v1.1.1)
- github.com/kr/text: [v0.1.0 → v0.2.0](https://github.com/kr/text/compare/v0.1.0...v0.2.0)
- github.com/kubernetes-csi/csi-lib-utils: [v0.9.1 → v0.10.0](https://github.com/kubernetes-csi/csi-lib-utils/compare/v0.9.1...v0.10.0)
- github.com/mattn/go-isatty: [v0.0.4 → v0.0.3](https://github.com/mattn/go-isatty/compare/v0.0.4...v0.0.3)
- github.com/mitchellh/go-homedir: [v1.0.0 → v1.1.0](https://github.com/mitchellh/go-homedir/compare/v1.0.0...v1.1.0)
- github.com/moby/term: [672ec06 → 9d4ed18](https://github.com/moby/term/compare/672ec06...9d4ed18)
- github.com/onsi/ginkgo: [v1.11.0 → v1.14.0](https://github.com/onsi/ginkgo/compare/v1.11.0...v1.14.0)
- github.com/onsi/gomega: [v1.7.1 → v1.10.1](https://github.com/onsi/gomega/compare/v1.7.1...v1.10.1)
- github.com/prometheus/client_golang: [v1.8.0 → v1.11.0](https://github.com/prometheus/client_golang/compare/v1.8.0...v1.11.0)
- github.com/prometheus/common: [v0.15.0 → v0.26.0](https://github.com/prometheus/common/compare/v0.15.0...v0.26.0)
- github.com/prometheus/procfs: [v0.2.0 → v0.6.0](https://github.com/prometheus/procfs/compare/v0.2.0...v0.6.0)
- github.com/rogpeppe/fastuuid: [6724a57 → v1.2.0](https://github.com/rogpeppe/fastuuid/compare/6724a57...v1.2.0)
- github.com/spf13/cobra: [v0.0.3 → v1.1.3](https://github.com/spf13/cobra/compare/v0.0.3...v1.1.3)
- github.com/stretchr/objx: [v0.2.0 → v0.1.1](https://github.com/stretchr/objx/compare/v0.2.0...v0.1.1)
- github.com/stretchr/testify: [v1.6.1 → v1.7.0](https://github.com/stretchr/testify/compare/v1.6.1...v1.7.0)
- github.com/tmc/grpc-websocket-proxy: [89b8d40 → 0ad062e](https://github.com/tmc/grpc-websocket-proxy/compare/89b8d40...0ad062e)
- github.com/yuin/goldmark: [v1.1.32 → v1.3.5](https://github.com/yuin/goldmark/compare/v1.1.32...v1.3.5)
- go.etcd.io/bbolt: v1.3.3 → v1.3.2
- go.uber.org/atomic: v1.5.0 → v1.7.0
- go.uber.org/multierr: v1.3.0 → v1.6.0
- go.uber.org/zap: v1.13.0 → v1.17.0
- golang.org/x/crypto: 5f87f34 → 5ea612d
- golang.org/x/lint: 738671d → 6edffad
- golang.org/x/mod: v0.3.0 → v0.4.2
- golang.org/x/net: 986b41b → 37e1c6a
- golang.org/x/oauth2: 08078c5 → cd4f82c
- golang.org/x/sync: 6e8e738 → 036812b
- golang.org/x/sys: f9fddec → 59db8d7
- golang.org/x/term: 2321bbc → de623e6
- golang.org/x/text: v0.3.4 → v0.3.6
- golang.org/x/time: 7e3f01d → 1f47c86
- golang.org/x/tools: b303f43 → v0.1.2
- google.golang.org/genproto: 8c77b98 → f16073e
- google.golang.org/grpc: v1.34.0 → v1.38.0
- google.golang.org/protobuf: v1.25.0 → v1.26.0
- gopkg.in/check.v1: 41f04d3 → 8fa4692
- gopkg.in/yaml.v3: eeeca48 → 496545a
- gotest.tools/v3: v3.0.2 → v3.0.3
- k8s.io/api: v0.20.0 → v0.22.0
- k8s.io/apimachinery: v0.20.0 → v0.22.0
- k8s.io/client-go: v0.20.0 → v0.22.0
- k8s.io/component-base: v0.20.0 → v0.22.0
- k8s.io/csi-translation-lib: v0.21.0-alpha.3 → v0.22.0
- k8s.io/klog/v2: v2.5.0 → v2.9.0
- k8s.io/kube-openapi: d219536 → 9528897
- k8s.io/utils: 67b214c → 4b05e18
- sigs.k8s.io/structured-merge-diff/v4: v4.0.2 → v4.1.2

### Removed
- github.com/Knetic/govaluate: [9aa4983](https://github.com/Knetic/govaluate/tree/9aa4983)
- github.com/Shopify/sarama: [v1.19.0](https://github.com/Shopify/sarama/tree/v1.19.0)
- github.com/Shopify/toxiproxy: [v2.1.4+incompatible](https://github.com/Shopify/toxiproxy/tree/v2.1.4)
- github.com/VividCortex/gohistogram: [v1.0.0](https://github.com/VividCortex/gohistogram/tree/v1.0.0)
- github.com/afex/hystrix-go: [fa1af6a](https://github.com/afex/hystrix-go/tree/fa1af6a)
- github.com/apache/thrift: [v0.13.0](https://github.com/apache/thrift/tree/v0.13.0)
- github.com/aryann/difflib: [e206f87](https://github.com/aryann/difflib/tree/e206f87)
- github.com/aws/aws-lambda-go: [v1.13.3](https://github.com/aws/aws-lambda-go/tree/v1.13.3)
- github.com/aws/aws-sdk-go-v2: [v0.18.0](https://github.com/aws/aws-sdk-go-v2/tree/v0.18.0)
- github.com/aws/aws-sdk-go: [v1.27.0](https://github.com/aws/aws-sdk-go/tree/v1.27.0)
- github.com/casbin/casbin/v2: [v2.1.2](https://github.com/casbin/casbin/v2/tree/v2.1.2)
- github.com/cenkalti/backoff: [v2.2.1+incompatible](https://github.com/cenkalti/backoff/tree/v2.2.1)
- github.com/clbanning/x2j: [8252494](https://github.com/clbanning/x2j/tree/8252494)
- github.com/cockroachdb/datadriven: [80d97fb](https://github.com/cockroachdb/datadriven/tree/80d97fb)
- github.com/codahale/hdrhistogram: [3a0bb77](https://github.com/codahale/hdrhistogram/tree/3a0bb77)
- github.com/docker/spdystream: [449fdfc](https://github.com/docker/spdystream/tree/449fdfc)
- github.com/dustin/go-humanize: [bb3d318](https://github.com/dustin/go-humanize/tree/bb3d318)
- github.com/eapache/go-resiliency: [v1.1.0](https://github.com/eapache/go-resiliency/tree/v1.1.0)
- github.com/eapache/go-xerial-snappy: [776d571](https://github.com/eapache/go-xerial-snappy/tree/776d571)
- github.com/eapache/queue: [v1.1.0](https://github.com/eapache/queue/tree/v1.1.0)
- github.com/edsrzf/mmap-go: [v1.0.0](https://github.com/edsrzf/mmap-go/tree/v1.0.0)
- github.com/franela/goblin: [c9ffbef](https://github.com/franela/goblin/tree/c9ffbef)
- github.com/franela/goreq: [bcd34c9](https://github.com/franela/goreq/tree/bcd34c9)
- github.com/go-openapi/spec: [v0.19.3](https://github.com/go-openapi/spec/tree/v0.19.3)
- github.com/go-sql-driver/mysql: [v1.4.0](https://github.com/go-sql-driver/mysql/tree/v1.4.0)
- github.com/gogo/googleapis: [v1.1.0](https://github.com/gogo/googleapis/tree/v1.1.0)
- github.com/golang/snappy: [2e65f85](https://github.com/golang/snappy/tree/2e65f85)
- github.com/gorilla/context: [v1.1.1](https://github.com/gorilla/context/tree/v1.1.1)
- github.com/gorilla/mux: [v1.7.3](https://github.com/gorilla/mux/tree/v1.7.3)
- github.com/hashicorp/go-version: [v1.2.0](https://github.com/hashicorp/go-version/tree/v1.2.0)
- github.com/hudl/fargo: [v1.3.0](https://github.com/hudl/fargo/tree/v1.3.0)
- github.com/influxdata/influxdb1-client: [8bf82d3](https://github.com/influxdata/influxdb1-client/tree/8bf82d3)
- github.com/jmespath/go-jmespath: [c2b33e8](https://github.com/jmespath/go-jmespath/tree/c2b33e8)
- github.com/lightstep/lightstep-tracer-common/golang/gogo: [bc2310a](https://github.com/lightstep/lightstep-tracer-common/golang/gogo/tree/bc2310a)
- github.com/lightstep/lightstep-tracer-go: [v0.18.1](https://github.com/lightstep/lightstep-tracer-go/tree/v0.18.1)
- github.com/lyft/protoc-gen-validate: [v0.0.13](https://github.com/lyft/protoc-gen-validate/tree/v0.0.13)
- github.com/mattn/go-runewidth: [v0.0.2](https://github.com/mattn/go-runewidth/tree/v0.0.2)
- github.com/nats-io/jwt: [v0.3.2](https://github.com/nats-io/jwt/tree/v0.3.2)
- github.com/nats-io/nats-server/v2: [v2.1.2](https://github.com/nats-io/nats-server/v2/tree/v2.1.2)
- github.com/nats-io/nats.go: [v1.9.1](https://github.com/nats-io/nats.go/tree/v1.9.1)
- github.com/nats-io/nkeys: [v0.1.3](https://github.com/nats-io/nkeys/tree/v0.1.3)
- github.com/nats-io/nuid: [v1.0.1](https://github.com/nats-io/nuid/tree/v1.0.1)
- github.com/oklog/oklog: [v0.3.2](https://github.com/oklog/oklog/tree/v0.3.2)
- github.com/oklog/run: [v1.0.0](https://github.com/oklog/run/tree/v1.0.0)
- github.com/olekukonko/tablewriter: [a0225b3](https://github.com/olekukonko/tablewriter/tree/a0225b3)
- github.com/op/go-logging: [970db52](https://github.com/op/go-logging/tree/970db52)
- github.com/opentracing-contrib/go-observer: [a52f234](https://github.com/opentracing-contrib/go-observer/tree/a52f234)
- github.com/opentracing/basictracer-go: [v1.0.0](https://github.com/opentracing/basictracer-go/tree/v1.0.0)
- github.com/opentracing/opentracing-go: [v1.1.0](https://github.com/opentracing/opentracing-go/tree/v1.1.0)
- github.com/openzipkin-contrib/zipkin-go-opentracing: [v0.4.5](https://github.com/openzipkin-contrib/zipkin-go-opentracing/tree/v0.4.5)
- github.com/openzipkin/zipkin-go: [v0.2.2](https://github.com/openzipkin/zipkin-go/tree/v0.2.2)
- github.com/pact-foundation/pact-go: [v1.0.4](https://github.com/pact-foundation/pact-go/tree/v1.0.4)
- github.com/pborman/uuid: [v1.2.0](https://github.com/pborman/uuid/tree/v1.2.0)
- github.com/performancecopilot/speed: [v3.0.0+incompatible](https://github.com/performancecopilot/speed/tree/v3.0.0)
- github.com/pierrec/lz4: [v2.0.5+incompatible](https://github.com/pierrec/lz4/tree/v2.0.5)
- github.com/pkg/profile: [v1.2.1](https://github.com/pkg/profile/tree/v1.2.1)
- github.com/rcrowley/go-metrics: [3113b84](https://github.com/rcrowley/go-metrics/tree/3113b84)
- github.com/samuel/go-zookeeper: [2cc03de](https://github.com/samuel/go-zookeeper/tree/2cc03de)
- github.com/sony/gobreaker: [v0.4.1](https://github.com/sony/gobreaker/tree/v0.4.1)
- github.com/streadway/amqp: [edfb901](https://github.com/streadway/amqp/tree/edfb901)
- github.com/streadway/handy: [d5acb31](https://github.com/streadway/handy/tree/d5acb31)
- github.com/urfave/cli: [v1.22.1](https://github.com/urfave/cli/tree/v1.22.1)
- go.etcd.io/etcd: 3cf2f69
- go.uber.org/tools: 2cfd321
- gopkg.in/cheggaaa/pb.v1: v1.0.25
- gopkg.in/gcfg.v1: v1.2.3
- gopkg.in/warnings.v0: v0.1.2
- gotest.tools: v2.2.0+incompatible
- sourcegraph.com/sourcegraph/appdash: ebfcffb
