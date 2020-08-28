# Release notes for v3.0.0

[Documentation](https://kubernetes-csi.github.io/docs/)

# Changelog since v2.2.0

## Urgent Upgrade Notes 

### (No, really, you MUST read this before you upgrade)

- Update volumeAttachment to v1
  
  RBAC policy was updated to allow the external-attacher to patch VolumeAttachment.Status ([#200](https://github.com/kubernetes-csi/external-attacher/pull/200), [@cwdsuzhou](https://github.com/cwdsuzhou))
 - Use GA version of CSINode object. The external-attacher now requires Kubernetes 1.17. ([#193](https://github.com/kubernetes-csi/external-attacher/pull/193), [@bertinatto](https://github.com/bertinatto))
 
## Changes by Kind

### Feature

- Added support for migration of Kubernetes in-tree VMware volumes to CSI. ([#236](https://github.com/kubernetes-csi/external-attacher/pull/236), [@divyenpatel](https://github.com/divyenpatel))

### Bug or Regression

- Fixes an issue in volume attachment reconciler when the CSI driver supports LIST_VOLUMES_PUBLISHED_NODES but does not implement CSI migration. ([#244](https://github.com/kubernetes-csi/external-attacher/pull/244), [@yuga711](https://github.com/yuga711))
- Use dedicated Kubernetes client for leader election that does not get throttled when the external-attacher is under heavy load. ([#242](https://github.com/kubernetes-csi/external-attacher/pull/242), [@jsafrane](https://github.com/jsafrane))

### Other (Cleanup or Flake)

- Removed support of go dep. ([#239](https://github.com/kubernetes-csi/external-attacher/pull/239), [@jsafrane](https://github.com/jsafrane))

### Uncategorized

- Build with Go 1.15 ([#246](https://github.com/kubernetes-csi/external-attacher/pull/246), [@pohly](https://github.com/pohly))
- Publishing of images on k8s.gcr.io ([#231](https://github.com/kubernetes-csi/external-attacher/pull/231), [@pohly](https://github.com/pohly))
- Updated client-go to v0.18 ([#221](https://github.com/kubernetes-csi/external-attacher/pull/221), [@humblec](https://github.com/humblec))

## Dependencies

### Added
- cloud.google.com/go/bigquery: v1.0.1
- cloud.google.com/go/datastore: v1.0.0
- cloud.google.com/go/pubsub: v1.0.1
- cloud.google.com/go/storage: v1.0.0
- dmitri.shuralyov.com/gpu/mtl: 666a987
- github.com/BurntSushi/xgb: [27f1227](https://github.com/BurntSushi/xgb/tree/27f1227)
- github.com/chzyer/logex: [v1.1.10](https://github.com/chzyer/logex/tree/v1.1.10)
- github.com/chzyer/readline: [2972be2](https://github.com/chzyer/readline/tree/2972be2)
- github.com/chzyer/test: [a1ea475](https://github.com/chzyer/test/tree/a1ea475)
- github.com/cncf/udpa/go: [269d4d4](https://github.com/cncf/udpa/go/tree/269d4d4)
- github.com/docopt/docopt-go: [ee0de3b](https://github.com/docopt/docopt-go/tree/ee0de3b)
- github.com/go-gl/glfw/v3.3/glfw: [12ad95a](https://github.com/go-gl/glfw/v3.3/glfw/tree/12ad95a)
- github.com/google/renameio: [v0.1.0](https://github.com/google/renameio/tree/v0.1.0)
- github.com/ianlancetaylor/demangle: [5e5cf60](https://github.com/ianlancetaylor/demangle/tree/5e5cf60)
- github.com/kubernetes-csi/csi-test/v3: [v3.1.0](https://github.com/kubernetes-csi/csi-test/v3/tree/v3.1.0)
- github.com/robertkrimen/otto: [c382bd3](https://github.com/robertkrimen/otto/tree/c382bd3)
- github.com/rogpeppe/go-internal: [v1.3.0](https://github.com/rogpeppe/go-internal/tree/v1.3.0)
- golang.org/x/image: cff245a
- golang.org/x/mobile: d2bd2a2
- golang.org/x/mod: c90efee
- golang.org/x/xerrors: 9bdfabe
- google.golang.org/protobuf: v1.24.0
- gopkg.in/errgo.v2: v2.1.0
- gopkg.in/sourcemap.v1: v1.0.5
- k8s.io/klog/v2: v2.2.0
- rsc.io/binaryregexp: v0.2.0
- rsc.io/quote/v3: v3.1.0
- rsc.io/sampler: v1.3.0
- sigs.k8s.io/structured-merge-diff/v4: v4.0.1

### Changed
- cloud.google.com/go: v0.38.0 → v0.51.0
- github.com/Azure/go-autorest/autorest/adal: [v0.5.0 → v0.8.2](https://github.com/Azure/go-autorest/autorest/adal/compare/v0.5.0...v0.8.2)
- github.com/Azure/go-autorest/autorest/date: [v0.1.0 → v0.2.0](https://github.com/Azure/go-autorest/autorest/date/compare/v0.1.0...v0.2.0)
- github.com/Azure/go-autorest/autorest/mocks: [v0.2.0 → v0.3.0](https://github.com/Azure/go-autorest/autorest/mocks/compare/v0.2.0...v0.3.0)
- github.com/Azure/go-autorest/autorest: [v0.9.0 → v0.9.6](https://github.com/Azure/go-autorest/autorest/compare/v0.9.0...v0.9.6)
- github.com/elazarl/goproxy: [c4fc265 → 947c36d](https://github.com/elazarl/goproxy/compare/c4fc265...947c36d)
- github.com/envoyproxy/go-control-plane: [5f8ba28 → v0.9.4](https://github.com/envoyproxy/go-control-plane/compare/5f8ba28...v0.9.4)
- github.com/evanphx/json-patch: [v4.5.0+incompatible → v4.9.0+incompatible](https://github.com/evanphx/json-patch/compare/v4.5.0...v4.9.0)
- github.com/fsnotify/fsnotify: [v1.4.7 → v1.4.9](https://github.com/fsnotify/fsnotify/compare/v1.4.7...v1.4.9)
- github.com/go-logr/logr: [v0.1.0 → v0.2.0](https://github.com/go-logr/logr/compare/v0.1.0...v0.2.0)
- github.com/gogo/protobuf: [65acae2 → v1.3.1](https://github.com/gogo/protobuf/compare/65acae2...v1.3.1)
- github.com/golang/groupcache: [5b532d6 → 215e871](https://github.com/golang/groupcache/compare/5b532d6...215e871)
- github.com/golang/mock: [v1.2.0 → v1.4.3](https://github.com/golang/mock/compare/v1.2.0...v1.4.3)
- github.com/golang/protobuf: [v1.3.2 → v1.4.2](https://github.com/golang/protobuf/compare/v1.3.2...v1.4.2)
- github.com/google/go-cmp: [v0.3.0 → v0.4.0](https://github.com/google/go-cmp/compare/v0.3.0...v0.4.0)
- github.com/google/gofuzz: [v1.0.0 → v1.1.0](https://github.com/google/gofuzz/compare/v1.0.0...v1.1.0)
- github.com/google/pprof: [3ea8567 → d4f498a](https://github.com/google/pprof/compare/3ea8567...d4f498a)
- github.com/googleapis/gax-go/v2: [v2.0.4 → v2.0.5](https://github.com/googleapis/gax-go/v2/compare/v2.0.4...v2.0.5)
- github.com/googleapis/gnostic: [v0.2.0 → v0.4.1](https://github.com/googleapis/gnostic/compare/v0.2.0...v0.4.1)
- github.com/imdario/mergo: [v0.3.7 → v0.3.9](https://github.com/imdario/mergo/compare/v0.3.7...v0.3.9)
- github.com/json-iterator/go: [v1.1.8 → v1.1.10](https://github.com/json-iterator/go/compare/v1.1.8...v1.1.10)
- github.com/jstemmer/go-junit-report: [af01ea7 → v0.9.1](https://github.com/jstemmer/go-junit-report/compare/af01ea7...v0.9.1)
- github.com/konsorten/go-windows-terminal-sequences: [v1.0.1 → v1.0.2](https://github.com/konsorten/go-windows-terminal-sequences/compare/v1.0.1...v1.0.2)
- github.com/kr/pretty: [v0.1.0 → v0.2.0](https://github.com/kr/pretty/compare/v0.1.0...v0.2.0)
- github.com/onsi/ginkgo: [v1.10.2 → v1.11.0](https://github.com/onsi/ginkgo/compare/v1.10.2...v1.11.0)
- github.com/onsi/gomega: [v1.7.0 → v1.7.1](https://github.com/onsi/gomega/compare/v1.7.0...v1.7.1)
- github.com/pkg/errors: [v0.8.1 → v0.9.1](https://github.com/pkg/errors/compare/v0.8.1...v0.9.1)
- github.com/sirupsen/logrus: [v1.2.0 → v1.4.2](https://github.com/sirupsen/logrus/compare/v1.2.0...v1.4.2)
- go.opencensus.io: v0.21.0 → v0.22.2
- golang.org/x/crypto: 60c769a → 75b2880
- golang.org/x/exp: 509febe → da58074
- golang.org/x/lint: d0100b6 → fdd1cda
- golang.org/x/net: c0dbc17 → ab34263
- golang.org/x/oauth2: 0f29369 → 858c2ad
- golang.org/x/sync: 1122301 → cd5d95a
- golang.org/x/sys: 0732a99 → ed371f2
- golang.org/x/text: v0.3.2 → v0.3.3
- golang.org/x/time: 9d24e82 → 555d28b
- golang.org/x/tools: 2c0ae70 → 7b8e75d
- google.golang.org/api: v0.4.0 → v0.15.0
- google.golang.org/appengine: v1.5.0 → v1.6.5
- google.golang.org/genproto: 5c49e3e → cb27e3a
- google.golang.org/grpc: v1.26.0 → v1.28.0
- gopkg.in/check.v1: 788fd78 → 41f04d3
- gopkg.in/yaml.v2: v2.2.4 → v2.2.8
- honnef.co/go/tools: ea95bdf → v0.0.1-2019.2.3
- k8s.io/api: v0.17.0 → v0.19.0
- k8s.io/apimachinery: v0.17.1-beta.0 → v0.19.0
- k8s.io/client-go: v0.17.0 → v0.19.0
- k8s.io/csi-translation-lib: v0.17.0 → v0.19.0
- k8s.io/gengo: 0689ccc → 3a45101
- k8s.io/kube-openapi: 30be4d1 → 6aeccd4
- k8s.io/utils: e782cd3 → d5654de
- sigs.k8s.io/yaml: v1.1.0 → v1.2.0

### Removed
- github.com/kubernetes-csi/csi-test: [v2.0.0+incompatible](https://github.com/kubernetes-csi/csi-test/tree/v2.0.0)
