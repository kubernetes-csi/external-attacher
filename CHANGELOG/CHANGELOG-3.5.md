# Release notes for v3.5.1

[Documentation](https://kubernetes-csi.github.io)

# Changelog since v3.5.0

## Changes by Kind

### Other (Cleanup or Flake)

- Updated dependencies to address CVE-2022-1996. ([#394](https://github.com/kubernetes-csi/external-attacher/pull/394), [@datamattsson](https://github.com/datamattsson))

## Dependencies

### Added
- cloud.google.com/go/firestore: v1.1.0
- github.com/armon/circbuf: [bbbad09](https://github.com/armon/circbuf/tree/bbbad09)
- github.com/armon/go-metrics: [f0300d1](https://github.com/armon/go-metrics/tree/f0300d1)
- github.com/armon/go-radix: [7fddfc3](https://github.com/armon/go-radix/tree/7fddfc3)
- github.com/bgentry/speakeasy: [v0.1.0](https://github.com/bgentry/speakeasy/tree/v0.1.0)
- github.com/bketelsen/crypt: [v0.0.4](https://github.com/bketelsen/crypt/tree/v0.0.4)
- github.com/blang/semver: [v3.5.1+incompatible](https://github.com/blang/semver/tree/v3.5.1)
- github.com/coreos/bbolt: [v1.3.2](https://github.com/coreos/bbolt/tree/v1.3.2)
- github.com/coreos/etcd: [v3.3.13+incompatible](https://github.com/coreos/etcd/tree/v3.3.13)
- github.com/coreos/go-semver: [v0.3.0](https://github.com/coreos/go-semver/tree/v0.3.0)
- github.com/coreos/go-systemd/v22: [v22.3.2](https://github.com/coreos/go-systemd/v22/tree/v22.3.2)
- github.com/coreos/go-systemd: [95778df](https://github.com/coreos/go-systemd/tree/95778df)
- github.com/coreos/pkg: [399ea9e](https://github.com/coreos/pkg/tree/399ea9e)
- github.com/dgrijalva/jwt-go: [v3.2.0+incompatible](https://github.com/dgrijalva/jwt-go/tree/v3.2.0)
- github.com/dgryski/go-sip13: [e10d5fe](https://github.com/dgryski/go-sip13/tree/e10d5fe)
- github.com/emicklei/go-restful/v3: [v3.8.0](https://github.com/emicklei/go-restful/v3/tree/v3.8.0)
- github.com/fatih/color: [v1.7.0](https://github.com/fatih/color/tree/v1.7.0)
- github.com/godbus/dbus/v5: [v5.0.4](https://github.com/godbus/dbus/v5/tree/v5.0.4)
- github.com/googleapis/gnostic: [v0.5.5](https://github.com/googleapis/gnostic/tree/v0.5.5)
- github.com/gopherjs/gopherjs: [0766667](https://github.com/gopherjs/gopherjs/tree/0766667)
- github.com/grpc-ecosystem/go-grpc-middleware: [v1.0.0](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/v1.0.0)
- github.com/grpc-ecosystem/go-grpc-prometheus: [v1.2.0](https://github.com/grpc-ecosystem/go-grpc-prometheus/tree/v1.2.0)
- github.com/hashicorp/consul/api: [v1.1.0](https://github.com/hashicorp/consul/api/tree/v1.1.0)
- github.com/hashicorp/consul/sdk: [v0.1.1](https://github.com/hashicorp/consul/sdk/tree/v0.1.1)
- github.com/hashicorp/errwrap: [v1.0.0](https://github.com/hashicorp/errwrap/tree/v1.0.0)
- github.com/hashicorp/go-cleanhttp: [v0.5.1](https://github.com/hashicorp/go-cleanhttp/tree/v0.5.1)
- github.com/hashicorp/go-immutable-radix: [v1.0.0](https://github.com/hashicorp/go-immutable-radix/tree/v1.0.0)
- github.com/hashicorp/go-msgpack: [v0.5.3](https://github.com/hashicorp/go-msgpack/tree/v0.5.3)
- github.com/hashicorp/go-multierror: [v1.0.0](https://github.com/hashicorp/go-multierror/tree/v1.0.0)
- github.com/hashicorp/go-rootcerts: [v1.0.0](https://github.com/hashicorp/go-rootcerts/tree/v1.0.0)
- github.com/hashicorp/go-sockaddr: [v1.0.0](https://github.com/hashicorp/go-sockaddr/tree/v1.0.0)
- github.com/hashicorp/go-syslog: [v1.0.0](https://github.com/hashicorp/go-syslog/tree/v1.0.0)
- github.com/hashicorp/go-uuid: [v1.0.1](https://github.com/hashicorp/go-uuid/tree/v1.0.1)
- github.com/hashicorp/go.net: [v0.0.1](https://github.com/hashicorp/go.net/tree/v0.0.1)
- github.com/hashicorp/hcl: [v1.0.0](https://github.com/hashicorp/hcl/tree/v1.0.0)
- github.com/hashicorp/logutils: [v1.0.0](https://github.com/hashicorp/logutils/tree/v1.0.0)
- github.com/hashicorp/mdns: [v1.0.0](https://github.com/hashicorp/mdns/tree/v1.0.0)
- github.com/hashicorp/memberlist: [v0.1.3](https://github.com/hashicorp/memberlist/tree/v0.1.3)
- github.com/hashicorp/serf: [v0.8.2](https://github.com/hashicorp/serf/tree/v0.8.2)
- github.com/jonboulle/clockwork: [v0.1.0](https://github.com/jonboulle/clockwork/tree/v0.1.0)
- github.com/jtolds/gls: [v4.20.0+incompatible](https://github.com/jtolds/gls/tree/v4.20.0)
- github.com/kr/fs: [v0.1.0](https://github.com/kr/fs/tree/v0.1.0)
- github.com/magiconair/properties: [v1.8.5](https://github.com/magiconair/properties/tree/v1.8.5)
- github.com/mattn/go-colorable: [v0.0.9](https://github.com/mattn/go-colorable/tree/v0.0.9)
- github.com/mattn/go-isatty: [v0.0.3](https://github.com/mattn/go-isatty/tree/v0.0.3)
- github.com/miekg/dns: [v1.0.14](https://github.com/miekg/dns/tree/v1.0.14)
- github.com/mitchellh/cli: [v1.0.0](https://github.com/mitchellh/cli/tree/v1.0.0)
- github.com/mitchellh/go-homedir: [v1.1.0](https://github.com/mitchellh/go-homedir/tree/v1.1.0)
- github.com/mitchellh/go-testing-interface: [v1.0.0](https://github.com/mitchellh/go-testing-interface/tree/v1.0.0)
- github.com/mitchellh/gox: [v0.4.0](https://github.com/mitchellh/gox/tree/v0.4.0)
- github.com/mitchellh/iochan: [v1.0.0](https://github.com/mitchellh/iochan/tree/v1.0.0)
- github.com/oklog/ulid: [v1.3.1](https://github.com/oklog/ulid/tree/v1.3.1)
- github.com/onsi/ginkgo/v2: [v2.1.4](https://github.com/onsi/ginkgo/v2/tree/v2.1.4)
- github.com/pascaldekloe/goe: [57f6aae](https://github.com/pascaldekloe/goe/tree/57f6aae)
- github.com/pelletier/go-toml: [v1.9.3](https://github.com/pelletier/go-toml/tree/v1.9.3)
- github.com/pkg/sftp: [v1.10.1](https://github.com/pkg/sftp/tree/v1.10.1)
- github.com/posener/complete: [v1.1.1](https://github.com/posener/complete/tree/v1.1.1)
- github.com/prometheus/tsdb: [v0.7.1](https://github.com/prometheus/tsdb/tree/v0.7.1)
- github.com/ryanuber/columnize: [9b3edd6](https://github.com/ryanuber/columnize/tree/9b3edd6)
- github.com/sean-/seed: [e2103e2](https://github.com/sean-/seed/tree/e2103e2)
- github.com/shurcooL/sanitized_anchor_name: [v1.0.0](https://github.com/shurcooL/sanitized_anchor_name/tree/v1.0.0)
- github.com/smartystreets/assertions: [b2de0cb](https://github.com/smartystreets/assertions/tree/b2de0cb)
- github.com/smartystreets/goconvey: [v1.6.4](https://github.com/smartystreets/goconvey/tree/v1.6.4)
- github.com/soheilhy/cmux: [v0.1.4](https://github.com/soheilhy/cmux/tree/v0.1.4)
- github.com/spf13/cast: [v1.3.1](https://github.com/spf13/cast/tree/v1.3.1)
- github.com/spf13/jwalterweatherman: [v1.1.0](https://github.com/spf13/jwalterweatherman/tree/v1.1.0)
- github.com/spf13/viper: [v1.8.1](https://github.com/spf13/viper/tree/v1.8.1)
- github.com/subosito/gotenv: [v1.2.0](https://github.com/subosito/gotenv/tree/v1.2.0)
- github.com/tmc/grpc-websocket-proxy: [0ad062e](https://github.com/tmc/grpc-websocket-proxy/tree/0ad062e)
- github.com/xiang90/probing: [43a291a](https://github.com/xiang90/probing/tree/43a291a)
- go.etcd.io/bbolt: v1.3.2
- go.etcd.io/etcd/api/v3: v3.5.0
- go.etcd.io/etcd/client/pkg/v3: v3.5.0
- go.etcd.io/etcd/client/v2: v2.305.0
- gopkg.in/ini.v1: v1.62.0
- gopkg.in/resty.v1: v1.12.0

### Changed
- github.com/cncf/udpa/go: [5459f2c → 04548b0](https://github.com/cncf/udpa/go/compare/5459f2c...04548b0)
- github.com/cncf/xds/go: [fbca930 → cb28da3](https://github.com/cncf/xds/go/compare/fbca930...cb28da3)
- github.com/cpuguy83/go-md2man/v2: [v2.0.1 → v2.0.0](https://github.com/cpuguy83/go-md2man/v2/compare/v2.0.1...v2.0.0)
- github.com/envoyproxy/go-control-plane: [63b5d3c → 49ff273](https://github.com/envoyproxy/go-control-plane/compare/63b5d3c...49ff273)
- github.com/go-logr/logr: [v1.2.0 → v1.2.3](https://github.com/go-logr/logr/compare/v1.2.0...v1.2.3)
- github.com/google/go-cmp: [v0.5.5 → v0.5.9](https://github.com/google/go-cmp/compare/v0.5.5...v0.5.9)
- github.com/imdario/mergo: [v0.3.12 → v0.3.13](https://github.com/imdario/mergo/compare/v0.3.12...v0.3.13)
- github.com/mitchellh/mapstructure: [v1.1.2 → v1.4.1](https://github.com/mitchellh/mapstructure/compare/v1.1.2...v1.4.1)
- github.com/moby/term: [3f7ff69 → 9d4ed18](https://github.com/moby/term/compare/3f7ff69...9d4ed18)
- github.com/onsi/gomega: [v1.10.1 → v1.19.0](https://github.com/onsi/gomega/compare/v1.10.1...v1.19.0)
- github.com/russross/blackfriday/v2: [v2.1.0 → v2.0.1](https://github.com/russross/blackfriday/v2/compare/v2.1.0...v2.0.1)
- github.com/spf13/afero: [v1.2.2 → v1.6.0](https://github.com/spf13/afero/compare/v1.2.2...v1.6.0)
- github.com/spf13/cobra: [v1.4.0 → v1.2.1](https://github.com/spf13/cobra/compare/v1.4.0...v1.2.1)
- github.com/stretchr/objx: [v0.1.1 → v0.4.0](https://github.com/stretchr/objx/compare/v0.1.1...v0.4.0)
- github.com/stretchr/testify: [v1.7.0 → v1.8.0](https://github.com/stretchr/testify/compare/v1.7.0...v1.8.0)
- github.com/yuin/goldmark: [v1.4.1 → v1.4.13](https://github.com/yuin/goldmark/compare/v1.4.1...v1.4.13)
- go.uber.org/goleak: v1.1.10 → v1.2.0
- golang.org/x/mod: 9b9b3d8 → 86c51ed
- golang.org/x/net: cd36cc0 → v0.4.0
- golang.org/x/sync: 036812b → 886fb93
- golang.org/x/sys: 3681064 → v0.3.0
- golang.org/x/term: 03fcf44 → v0.3.0
- golang.org/x/text: v0.3.7 → v0.5.0
- golang.org/x/tools: 897bd77 → v0.1.12
- google.golang.org/api: v0.43.0 → v0.44.0
- google.golang.org/grpc: v1.40.0 → v1.51.0
- google.golang.org/protobuf: v1.27.1 → v1.28.1
- k8s.io/api: v0.24.0 → v0.24.8
- k8s.io/apimachinery: v0.24.0 → v0.24.8
- k8s.io/client-go: v0.24.0 → v0.24.8
- k8s.io/component-base: v0.24.0 → v0.23.14
- k8s.io/csi-translation-lib: v0.24.0 → v0.24.8
- k8s.io/klog/v2: v2.60.1 → v2.80.1
- k8s.io/kube-openapi: 3ee0da9 → 172d655
- k8s.io/utils: 3a6ce19 → 1a15be2
- sigs.k8s.io/json: 9f7c6b3 → f223a00
- sigs.k8s.io/structured-merge-diff/v4: v4.2.1 → v4.2.3
- sigs.k8s.io/yaml: v1.2.0 → v1.3.0

### Removed
- github.com/blang/semver/v4: [v4.0.0](https://github.com/blang/semver/v4/tree/v4.0.0)
hub.com/mitchellh/gox/tree/v0.4.0)
- github.com/mitchellh/iochan: [v1.0.0](https://github.com/mitchellh/iochan/tree/v1.0.0)
- github.com/oklog/ulid: [v1.3.1](https://github.com/oklog/ulid/tree/v1.3.1)
- github.com/pascaldekloe/goe: [57f6aae](https://github.com/pascaldekloe/goe/tree/57f6aae)
- github.com/pelletier/go-toml: [v1.2.0](https://github.com/pelletier/go-toml/tree/v1.2.0)
- github.com/posener/complete: [v1.1.1](https://github.com/posener/complete/tree/v1.1.1)
- github.com/prometheus/tsdb: [v0.7.1](https://github.com/prometheus/tsdb/tree/v0.7.1)
- github.com/ryanuber/columnize: [9b3edd6](https://github.com/ryanuber/columnize/tree/9b3edd6)
- github.com/sean-/seed: [e2103e2](https://github.com/sean-/seed/tree/e2103e2)
- github.com/shurcooL/sanitized_anchor_name: [v1.0.0](https://github.com/shurcooL/sanitized_anchor_name/tree/v1.0.0)
- github.com/smartystreets/assertions: [b2de0cb](https://github.com/smartystreets/assertions/tree/b2de0cb)
- github.com/smartystreets/goconvey: [v1.6.4](https://github.com/smartystreets/goconvey/tree/v1.6.4)
- github.com/soheilhy/cmux: [v0.1.4](https://github.com/soheilhy/cmux/tree/v0.1.4)
- github.com/spf13/cast: [v1.3.0](https://github.com/spf13/cast/tree/v1.3.0)
- github.com/spf13/jwalterweatherman: [v1.0.0](https://github.com/spf13/jwalterweatherman/tree/v1.0.0)
- github.com/spf13/viper: [v1.7.0](https://github.com/spf13/viper/tree/v1.7.0)
- github.com/subosito/gotenv: [v1.2.0](https://github.com/subosito/gotenv/tree/v1.2.0)
- github.com/tmc/grpc-websocket-proxy: [0ad062e](https://github.com/tmc/grpc-websocket-proxy/tree/0ad062e)
- github.com/xeipuuv/gojsonpointer: [4e3ac27](https://github.com/xeipuuv/gojsonpointer/tree/4e3ac27)
- github.com/xeipuuv/gojsonreference: [bd5ef7b](https://github.com/xeipuuv/gojsonreference/tree/bd5ef7b)
- github.com/xeipuuv/gojsonschema: [v1.2.0](https://github.com/xeipuuv/gojsonschema/tree/v1.2.0)
- github.com/xiang90/probing: [43a291a](https://github.com/xiang90/probing/tree/43a291a)
- go.etcd.io/bbolt: v1.3.2
- go.opentelemetry.io/otel/exporters/otlp/internal/retry: v1.10.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc: v1.10.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace: v1.10.0
- gopkg.in/ini.v1: v1.51.0
- gopkg.in/resty.v1: v1.12.0

### Changed
- cloud.google.com/go/bigquery: v1.8.0 → v1.43.0
- cloud.google.com/go: v0.97.0 → v0.105.0
- github.com/Azure/go-autorest/autorest/adal: [v0.9.20 → v0.9.13](https://github.com/Azure/go-autorest/autorest/adal/compare/v0.9.20...v0.9.13)
- github.com/Azure/go-autorest/autorest/mocks: [v0.4.2 → v0.4.1](https://github.com/Azure/go-autorest/autorest/mocks/compare/v0.4.2...v0.4.1)
- github.com/Azure/go-autorest/autorest: [v0.11.27 → v0.11.18](https://github.com/Azure/go-autorest/autorest/compare/v0.11.27...v0.11.18)
- github.com/benbjohnson/clock: [v1.1.0 → v1.0.3](https://github.com/benbjohnson/clock/compare/v1.1.0...v1.0.3)
- github.com/container-storage-interface/spec: [v1.5.0 → v1.7.0](https://github.com/container-storage-interface/spec/compare/v1.5.0...v1.7.0)
- github.com/cpuguy83/go-md2man/v2: [v2.0.1 → v2.0.0](https://github.com/cpuguy83/go-md2man/v2/compare/v2.0.1...v2.0.0)
- github.com/emicklei/go-restful/v3: [v3.8.0 → v3.10.0](https://github.com/emicklei/go-restful/v3/compare/v3.8.0...v3.10.0)
- github.com/evanphx/json-patch: [v4.12.0+incompatible → v5.6.0+incompatible](https://github.com/evanphx/json-patch/compare/v4.12.0...v5.6.0)
- github.com/felixge/httpsnoop: [v1.0.1 → v1.0.3](https://github.com/felixge/httpsnoop/compare/v1.0.1...v1.0.3)
- github.com/go-kit/log: [v0.1.0 → v0.2.0](https://github.com/go-kit/log/compare/v0.1.0...v0.2.0)
- github.com/go-logfmt/logfmt: [v0.5.0 → v0.5.1](https://github.com/go-logfmt/logfmt/compare/v0.5.0...v0.5.1)
- github.com/go-openapi/jsonreference: [v0.19.5 → v0.20.0](https://github.com/go-openapi/jsonreference/compare/v0.19.5...v0.20.0)
- github.com/go-openapi/swag: [v0.19.14 → v0.22.3](https://github.com/go-openapi/swag/compare/v0.19.14...v0.22.3)
- github.com/google/gnostic: [v0.5.7-v3refs → v0.6.9](https://github.com/google/gnostic/compare/v0.5.7-v3refs...v0.6.9)
- github.com/google/go-cmp: [v0.5.6 → v0.5.9](https://github.com/google/go-cmp/compare/v0.5.6...v0.5.9)
- github.com/google/martian/v3: [v3.2.1 → v3.0.0](https://github.com/google/martian/v3/compare/v3.2.1...v3.0.0)
- github.com/google/pprof: [4bb14d4 → 94a9f03](https://github.com/google/pprof/compare/4bb14d4...94a9f03)
- github.com/google/uuid: [v1.1.2 → v1.3.0](https://github.com/google/uuid/compare/v1.1.2...v1.3.0)
- github.com/googleapis/gax-go/v2: [v2.1.0 → v2.0.5](https://github.com/googleapis/gax-go/v2/compare/v2.1.0...v2.0.5)
- github.com/imdario/mergo: [v0.3.12 → v0.3.13](https://github.com/imdario/mergo/compare/v0.3.12...v0.3.13)
- github.com/inconshreveable/mousetrap: [v1.0.0 → v1.0.1](https://github.com/inconshreveable/mousetrap/compare/v1.0.0...v1.0.1)
- github.com/kubernetes-csi/csi-test/v4: [v4.0.2 → v4.4.0](https://github.com/kubernetes-csi/csi-test/v4/compare/v4.0.2...v4.4.0)
- github.com/mailru/easyjson: [v0.7.6 → v0.7.7](https://github.com/mailru/easyjson/compare/v0.7.6...v0.7.7)
- github.com/matttproud/golang_protobuf_extensions: [v1.0.1 → v1.0.4](https://github.com/matttproud/golang_protobuf_extensions/compare/v1.0.1...v1.0.4)
- github.com/moby/term: [3f7ff69 → 39b0c02](https://github.com/moby/term/compare/3f7ff69...39b0c02)
- github.com/onsi/ginkgo/v2: [v2.1.4 → v2.4.0](https://github.com/onsi/ginkgo/v2/compare/v2.1.4...v2.4.0)
- github.com/onsi/ginkgo: [v1.16.4 → v1.16.5](https://github.com/onsi/ginkgo/compare/v1.16.4...v1.16.5)
- github.com/onsi/gomega: [v1.19.0 → v1.23.0](https://github.com/onsi/gomega/compare/v1.19.0...v1.23.0)
- github.com/prometheus/client_golang: [v1.12.1 → v1.14.0](https://github.com/prometheus/client_golang/compare/v1.12.1...v1.14.0)
- github.com/prometheus/client_model: [v0.2.0 → v0.3.0](https://github.com/prometheus/client_model/compare/v0.2.0...v0.3.0)
- github.com/prometheus/common: [v0.32.1 → v0.37.0](https://github.com/prometheus/common/compare/v0.32.1...v0.37.0)
- github.com/prometheus/procfs: [v0.7.3 → v0.8.0](https://github.com/prometheus/procfs/compare/v0.7.3...v0.8.0)
- github.com/russross/blackfriday/v2: [v2.1.0 → v2.0.1](https://github.com/russross/blackfriday/v2/compare/v2.1.0...v2.0.1)
- github.com/spf13/cobra: [v1.4.0 → v1.6.0](https://github.com/spf13/cobra/compare/v1.4.0...v1.6.0)
- github.com/stretchr/testify: [v1.7.0 → v1.8.0](https://github.com/stretchr/testify/compare/v1.7.0...v1.8.0)
- github.com/yuin/goldmark: [v1.4.13 → v1.3.5](https://github.com/yuin/goldmark/compare/v1.4.13...v1.3.5)
- go.opencensus.io: v0.23.0 → v0.22.4
- go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp: v0.20.0 → v0.35.0
- go.opentelemetry.io/otel/metric: v0.20.0 → v0.31.0
- go.opentelemetry.io/otel/sdk: v0.20.0 → v1.10.0
- go.opentelemetry.io/otel/trace: v0.20.0 → v1.10.0
- go.opentelemetry.io/otel: v0.20.0 → v1.10.0
- go.opentelemetry.io/proto/otlp: v0.7.0 → v0.19.0
- go.uber.org/goleak: v1.1.10 → v1.2.0
- golang.org/x/crypto: 3147a52 → 5ea612d
- golang.org/x/net: a158d28 → 1e63c2f
- golang.org/x/oauth2: d3ed0bb → v0.2.0
- golang.org/x/sync: 886fb93 → 0de741c
- golang.org/x/sys: 8c9f86f → v0.3.0
- golang.org/x/term: 03fcf44 → v0.3.0
- golang.org/x/text: v0.3.7 → v0.5.0
- golang.org/x/time: 90d013b → v0.2.0
- google.golang.org/api: v0.57.0 → v0.30.0
- google.golang.org/genproto: c8bf987 → 142d8a6
- google.golang.org/grpc: v1.47.0 → v1.51.0
- google.golang.org/protobuf: v1.28.0 → v1.28.1
- gopkg.in/check.v1: 8fa4692 → 10cb982
- k8s.io/api: v0.25.0 → v0.26.0
- k8s.io/apimachinery: v0.25.0 → v0.26.0
- k8s.io/client-go: v0.25.0 → v0.26.0
- k8s.io/component-base: v0.25.0 → v0.26.0
- k8s.io/csi-translation-lib: v0.25.0 → v0.26.0
- k8s.io/klog/v2: v2.70.1 → v2.80.1
- k8s.io/kube-openapi: 67bda5d → a28e98e
- k8s.io/utils: ee6ede2 → 8e77b1f
- sigs.k8s.io/yaml: v1.2.0 → v1.3.0

### Removed
- github.com/getkin/kin-openapi: [v0.76.0](https://github.com/getkin/kin-openapi/tree/v0.76.0)
- github.com/golang-jwt/jwt/v4: [v4.2.0](https://github.com/golang-jwt/jwt/v4/tree/v4.2.0)
- github.com/golang/snappy: [v0.0.3](https://github.com/golang/snappy/tree/v0.0.3)
- github.com/gorilla/mux: [v1.8.0](https://github.com/gorilla/mux/tree/v1.8.0)
- github.com/robertkrimen/otto: [c382bd3](https://github.com/robertkrimen/otto/tree/c382bd3)
- google.golang.org/grpc/cmd/protoc-gen-go-grpc: v1.1.0
- gopkg.in/sourcemap.v1: v1.0.5
- k8s.io/klog: v1.0.0

# Release notes for v3.5.0

[Documentation](https://kubernetes-csi.github.io)
# Changelog since v3.4.0

## Changes by Kind

### Bug or Regression

- There is a new --default-fstype flag available in this release that defaults to ext4 and can be used to set any value a driver may need. In the next major release ext4 as the default fsType will be deprecated and replaced by "" ([#342](https://github.com/kubernetes-csi/external-attacher/pull/342), [@RomanBednar](https://github.com/RomanBednar))
- Upgrade csi-translation-lib for some bug fixes with azuredisk and azurefile ([#352](https://github.com/kubernetes-csi/external-attacher/pull/352), [@andyzhangx](https://github.com/andyzhangx))

### Uncategorized

- Kube client dependencies are updated to v1.24.0 ([#353](https://github.com/kubernetes-csi/external-attacher/pull/353), [@humblec](https://github.com/humblec))

## Dependencies

### Added
- github.com/armon/go-socks5: [e753329](https://github.com/armon/go-socks5/tree/e753329)
- github.com/blang/semver/v4: [v4.0.0](https://github.com/blang/semver/v4/tree/v4.0.0)
- github.com/google/gnostic: [v0.5.7-v3refs](https://github.com/google/gnostic/tree/v0.5.7-v3refs)
- github.com/josharian/intern: [v1.0.0](https://github.com/josharian/intern/tree/v1.0.0)

### Changed
- github.com/cespare/xxhash/v2: [v2.1.1 → v2.1.2](https://github.com/cespare/xxhash/v2/compare/v2.1.1...v2.1.2)
- github.com/cpuguy83/go-md2man/v2: [v2.0.0 → v2.0.1](https://github.com/cpuguy83/go-md2man/v2/compare/v2.0.0...v2.0.1)
- github.com/emicklei/go-restful: [ff4f55a → v2.9.5+incompatible](https://github.com/emicklei/go-restful/compare/ff4f55a...v2.9.5)
- github.com/go-openapi/jsonreference: [v0.19.3 → v0.19.5](https://github.com/go-openapi/jsonreference/compare/v0.19.3...v0.19.5)
- github.com/go-openapi/swag: [v0.19.5 → v0.19.14](https://github.com/go-openapi/swag/compare/v0.19.5...v0.19.14)
- github.com/kubernetes-csi/csi-lib-utils: [v0.10.0 → v0.11.0](https://github.com/kubernetes-csi/csi-lib-utils/compare/v0.10.0...v0.11.0)
- github.com/mailru/easyjson: [b2ccc51 → v0.7.6](https://github.com/mailru/easyjson/compare/b2ccc51...v0.7.6)
- github.com/mitchellh/mapstructure: [v1.4.1 → v1.1.2](https://github.com/mitchellh/mapstructure/compare/v1.4.1...v1.1.2)
- github.com/moby/term: [9d4ed18 → 3f7ff69](https://github.com/moby/term/compare/9d4ed18...3f7ff69)
- github.com/munnerz/goautoneg: [a547fc6 → a7dc8b6](https://github.com/munnerz/goautoneg/compare/a547fc6...a7dc8b6)
- github.com/prometheus/client_golang: [v1.11.0 → v1.12.1](https://github.com/prometheus/client_golang/compare/v1.11.0...v1.12.1)
- github.com/prometheus/common: [v0.28.0 → v0.32.1](https://github.com/prometheus/common/compare/v0.28.0...v0.32.1)
- github.com/prometheus/procfs: [v0.6.0 → v0.7.3](https://github.com/prometheus/procfs/compare/v0.6.0...v0.7.3)
- github.com/russross/blackfriday/v2: [v2.0.1 → v2.1.0](https://github.com/russross/blackfriday/v2/compare/v2.0.1...v2.1.0)
- github.com/spf13/afero: [v1.6.0 → v1.2.2](https://github.com/spf13/afero/compare/v1.6.0...v1.2.2)
- github.com/spf13/cobra: [v1.2.1 → v1.4.0](https://github.com/spf13/cobra/compare/v1.2.1...v1.4.0)
- github.com/yuin/goldmark: [v1.4.0 → v1.4.1](https://github.com/yuin/goldmark/compare/v1.4.0...v1.4.1)
- golang.org/x/crypto: 32db794 → 8634188
- golang.org/x/mod: v0.4.2 → 9b9b3d8
- golang.org/x/net: e898025 → cd36cc0
- golang.org/x/oauth2: 2bc19b1 → d3ed0bb
- golang.org/x/sys: f4d4317 → 3681064
- golang.org/x/term: 6886f2d → 03fcf44
- golang.org/x/time: 1f47c86 → 90d013b
- golang.org/x/tools: d4cc65f → 897bd77
- google.golang.org/api: v0.44.0 → v0.43.0
- google.golang.org/genproto: fe13028 → 42d7afd
- gopkg.in/yaml.v3: 496545a → v3.0.1
- k8s.io/api: v0.23.0 → v0.24.0
- k8s.io/apimachinery: v0.23.0 → v0.24.0
- k8s.io/client-go: v0.23.0 → v0.24.0
- k8s.io/component-base: v0.23.0 → v0.24.0
- k8s.io/csi-translation-lib: v0.23.0 → v0.24.0
- k8s.io/klog/v2: v2.30.0 → v2.60.1
- k8s.io/kube-openapi: e816edb → 3ee0da9
- k8s.io/utils: cb0fa31 → 3a6ce19
- sigs.k8s.io/json: c049b76 → 9f7c6b3
- sigs.k8s.io/structured-merge-diff/v4: v4.1.2 → v4.2.1

### Removed
- cloud.google.com/go/firestore: v1.1.0
- github.com/armon/circbuf: [bbbad09](https://github.com/armon/circbuf/tree/bbbad09)
- github.com/armon/go-metrics: [f0300d1](https://github.com/armon/go-metrics/tree/f0300d1)
- github.com/armon/go-radix: [7fddfc3](https://github.com/armon/go-radix/tree/7fddfc3)
- github.com/bgentry/speakeasy: [v0.1.0](https://github.com/bgentry/speakeasy/tree/v0.1.0)
- github.com/bketelsen/crypt: [v0.0.4](https://github.com/bketelsen/crypt/tree/v0.0.4)
- github.com/blang/semver: [v3.5.1+incompatible](https://github.com/blang/semver/tree/v3.5.1)
- github.com/coreos/go-semver: [v0.3.0](https://github.com/coreos/go-semver/tree/v0.3.0)
- github.com/coreos/go-systemd/v22: [v22.3.2](https://github.com/coreos/go-systemd/v22/tree/v22.3.2)
- github.com/fatih/color: [v1.7.0](https://github.com/fatih/color/tree/v1.7.0)
- github.com/godbus/dbus/v5: [v5.0.4](https://github.com/godbus/dbus/v5/tree/v5.0.4)
- github.com/googleapis/gnostic: [v0.5.5](https://github.com/googleapis/gnostic/tree/v0.5.5)
- github.com/gopherjs/gopherjs: [0766667](https://github.com/gopherjs/gopherjs/tree/0766667)
- github.com/hashicorp/consul/api: [v1.1.0](https://github.com/hashicorp/consul/api/tree/v1.1.0)
- github.com/hashicorp/consul/sdk: [v0.1.1](https://github.com/hashicorp/consul/sdk/tree/v0.1.1)
- github.com/hashicorp/errwrap: [v1.0.0](https://github.com/hashicorp/errwrap/tree/v1.0.0)
- github.com/hashicorp/go-cleanhttp: [v0.5.1](https://github.com/hashicorp/go-cleanhttp/tree/v0.5.1)
- github.com/hashicorp/go-immutable-radix: [v1.0.0](https://github.com/hashicorp/go-immutable-radix/tree/v1.0.0)
- github.com/hashicorp/go-msgpack: [v0.5.3](https://github.com/hashicorp/go-msgpack/tree/v0.5.3)
- github.com/hashicorp/go-multierror: [v1.0.0](https://github.com/hashicorp/go-multierror/tree/v1.0.0)
- github.com/hashicorp/go-rootcerts: [v1.0.0](https://github.com/hashicorp/go-rootcerts/tree/v1.0.0)
- github.com/hashicorp/go-sockaddr: [v1.0.0](https://github.com/hashicorp/go-sockaddr/tree/v1.0.0)
- github.com/hashicorp/go-syslog: [v1.0.0](https://github.com/hashicorp/go-syslog/tree/v1.0.0)
- github.com/hashicorp/go-uuid: [v1.0.1](https://github.com/hashicorp/go-uuid/tree/v1.0.1)
- github.com/hashicorp/go.net: [v0.0.1](https://github.com/hashicorp/go.net/tree/v0.0.1)
- github.com/hashicorp/hcl: [v1.0.0](https://github.com/hashicorp/hcl/tree/v1.0.0)
- github.com/hashicorp/logutils: [v1.0.0](https://github.com/hashicorp/logutils/tree/v1.0.0)
- github.com/hashicorp/mdns: [v1.0.0](https://github.com/hashicorp/mdns/tree/v1.0.0)
- github.com/hashicorp/memberlist: [v0.1.3](https://github.com/hashicorp/memberlist/tree/v0.1.3)
- github.com/hashicorp/serf: [v0.8.2](https://github.com/hashicorp/serf/tree/v0.8.2)
- github.com/jtolds/gls: [v4.20.0+incompatible](https://github.com/jtolds/gls/tree/v4.20.0)
- github.com/kr/fs: [v0.1.0](https://github.com/kr/fs/tree/v0.1.0)
- github.com/magiconair/properties: [v1.8.5](https://github.com/magiconair/properties/tree/v1.8.5)
- github.com/mattn/go-colorable: [v0.0.9](https://github.com/mattn/go-colorable/tree/v0.0.9)
- github.com/mattn/go-isatty: [v0.0.3](https://github.com/mattn/go-isatty/tree/v0.0.3)
- github.com/miekg/dns: [v1.0.14](https://github.com/miekg/dns/tree/v1.0.14)
- github.com/mitchellh/cli: [v1.0.0](https://github.com/mitchellh/cli/tree/v1.0.0)
- github.com/mitchellh/go-homedir: [v1.0.0](https://github.com/mitchellh/go-homedir/tree/v1.0.0)
- github.com/mitchellh/go-testing-interface: [v1.0.0](https://github.com/mitchellh/go-testing-interface/tree/v1.0.0)
- github.com/mitchellh/gox: [v0.4.0](https://github.com/mitchellh/gox/tree/v0.4.0)
- github.com/mitchellh/iochan: [v1.0.0](https://github.com/mitchellh/iochan/tree/v1.0.0)
- github.com/pascaldekloe/goe: [57f6aae](https://github.com/pascaldekloe/goe/tree/57f6aae)
- github.com/pelletier/go-toml: [v1.9.3](https://github.com/pelletier/go-toml/tree/v1.9.3)
- github.com/pkg/sftp: [v1.10.1](https://github.com/pkg/sftp/tree/v1.10.1)
- github.com/posener/complete: [v1.1.1](https://github.com/posener/complete/tree/v1.1.1)
- github.com/ryanuber/columnize: [9b3edd6](https://github.com/ryanuber/columnize/tree/9b3edd6)
- github.com/sean-/seed: [e2103e2](https://github.com/sean-/seed/tree/e2103e2)
- github.com/shurcooL/sanitized_anchor_name: [v1.0.0](https://github.com/shurcooL/sanitized_anchor_name/tree/v1.0.0)
- github.com/smartystreets/assertions: [b2de0cb](https://github.com/smartystreets/assertions/tree/b2de0cb)
- github.com/smartystreets/goconvey: [v1.6.4](https://github.com/smartystreets/goconvey/tree/v1.6.4)
- github.com/spf13/cast: [v1.3.1](https://github.com/spf13/cast/tree/v1.3.1)
- github.com/spf13/jwalterweatherman: [v1.1.0](https://github.com/spf13/jwalterweatherman/tree/v1.1.0)
- github.com/spf13/viper: [v1.8.1](https://github.com/spf13/viper/tree/v1.8.1)
- github.com/subosito/gotenv: [v1.2.0](https://github.com/subosito/gotenv/tree/v1.2.0)
- go.etcd.io/etcd/api/v3: v3.5.0
- go.etcd.io/etcd/client/pkg/v3: v3.5.0
- go.etcd.io/etcd/client/v2: v2.305.0
- gopkg.in/ini.v1: v1.62.0
