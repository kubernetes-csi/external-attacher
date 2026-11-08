module github.com/kubernetes-csi/external-attacher/v4

go 1.27.0

require (
	github.com/container-storage-interface/spec v1.14.0-rc1
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/evanphx/json-patch v5.9.11+incompatible
	github.com/kubernetes-csi/csi-lib-utils v0.25.0
	github.com/kubernetes-csi/csi-test/v5 v5.6.0
	go.uber.org/mock v0.6.0
	google.golang.org/grpc v1.85.0
	google.golang.org/protobuf v1.36.12
	k8s.io/api v0.38.0-beta.0
	k8s.io/apimachinery v0.38.0-beta.0
	k8s.io/apiserver v0.38.0-beta.0
	k8s.io/client-go v0.38.0-beta.0
	k8s.io/component-base v0.38.0-beta.0
	k8s.io/csi-translation-lib v0.38.0-beta.0
	k8s.io/klog/v2 v2.140.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fxamacker/cbor/v2 v2.10.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v1.0.2 // indirect
	github.com/go-openapi/jsonreference v1.0.3 // indirect
	github.com/go-openapi/swag/conv v0.29.2 // indirect
	github.com/go-openapi/swag/jsonutils v0.29.2 // indirect
	github.com/go-openapi/swag/pools v0.29.2 // indirect
	github.com/go-openapi/swag/typeutils v0.29.2 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pandatix/go-abnf v0.5.5 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.25.0 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.73.1 // indirect
	github.com/prometheus/procfs v0.23.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.1.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.72.0 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v2 v2.4.5 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/oauth2 v0.37.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/term v0.47.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261019180522-ce908d29299b // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/kube-openapi v0.0.0-20261014044854-ca3452a79603 // indirect
	k8s.io/utils v0.0.0-20260707023825-cf1189d6abe3 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v7 v7.0.1 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace (
	k8s.io/api => /Users/huww98/source/kubernetes/staging/src/k8s.io/api
	k8s.io/apimachinery => /Users/huww98/source/kubernetes/staging/src/k8s.io/apimachinery
	k8s.io/apiserver => /Users/huww98/source/kubernetes/staging/src/k8s.io/apiserver
	k8s.io/client-go => /Users/huww98/source/kubernetes/staging/src/k8s.io/client-go
	k8s.io/component-base => /Users/huww98/source/kubernetes/staging/src/k8s.io/component-base
	k8s.io/csi-translation-lib => /Users/huww98/source/kubernetes/staging/src/k8s.io/csi-translation-lib
	k8s.io/ktesting => /Users/huww98/source/kubernetes/staging/src/k8s.io/ktesting
)
