module github.com/kubernetes-csi/external-attacher

go 1.16

require (
	github.com/container-storage-interface/spec v1.5.0
	github.com/davecgh/go-spew v1.1.1
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.5.1
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.4 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/kubernetes-csi/csi-lib-utils v0.9.1
	github.com/kubernetes-csi/csi-test/v4 v4.0.2
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/prometheus/common v0.19.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	golang.org/x/net v0.0.0-20210410081132-afb366fc7cd1 // indirect
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84 // indirect
	golang.org/x/term v0.0.0-20210317153231-de623e64d2a6 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210317182105-75c7a8546eb9 // indirect
	google.golang.org/grpc v1.37.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.1
	k8s.io/csi-translation-lib v0.21.0
	k8s.io/klog/v2 v2.8.0
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10 // indirect
)

// go get -d github.com/chrishenzie/csi-lib-utils@single-node-access-modes
replace github.com/kubernetes-csi/csi-lib-utils => github.com/chrishenzie/csi-lib-utils v0.9.2-0.20210614221230-48c8713d1279

replace k8s.io/component-base => github.com/chrishenzie/kubernetes/staging/src/k8s.io/component-base v0.0.0-20210507180302-a29b4b67ec78

replace k8s.io/node-api => github.com/chrishenzie/kubernetes/staging/src/k8s.io/node-api v0.0.0-20210507180302-a29b4b67ec78

// go get -d github.com/chrishenzie/kubernetes/staging/src/k8s.io/api@read-write-once-pod-access-mode
replace k8s.io/api => github.com/chrishenzie/kubernetes/staging/src/k8s.io/api v0.0.0-20210507180302-a29b4b67ec78

replace k8s.io/apimachinery => github.com/chrishenzie/kubernetes/staging/src/k8s.io/apimachinery v0.0.0-20210507180302-a29b4b67ec78

replace k8s.io/client-go => github.com/chrishenzie/kubernetes/staging/src/k8s.io/client-go v0.0.0-20210507180302-a29b4b67ec78

replace k8s.io/csi-translation-lib => github.com/chrishenzie/kubernetes/staging/src/k8s.io/csi-translation-lib v0.0.0-20210507180302-a29b4b67ec78
