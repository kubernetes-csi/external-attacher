module github.com/kubernetes-csi/external-attacher

go 1.15

require (
	github.com/container-storage-interface/spec v1.5.0
	github.com/davecgh/go-spew v1.1.1
	github.com/evanphx/json-patch v4.12.0+incompatible
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/kubernetes-csi/csi-lib-utils v0.10.0
	github.com/kubernetes-csi/csi-test/v4 v4.3.0
	github.com/prometheus/client_golang v1.12.0 // indirect
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce // indirect
	golang.org/x/net v0.0.0-20220121210141-e204ce36a2ba // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220118154757-00ab72f36ad5 // indirect
	google.golang.org/grpc v1.43.0
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.22.0
	k8s.io/csi-translation-lib v0.20.0
	k8s.io/klog/v2 v2.40.1
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf // indirect
	k8s.io/utils v0.0.0-20211208161948-7d6a63dca704 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.5

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.0

replace k8s.io/apiserver => k8s.io/apiserver v0.20.0

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.0

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.0

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.0

replace k8s.io/code-generator => k8s.io/code-generator v0.20.0

replace k8s.io/component-base => k8s.io/component-base v0.20.0

replace k8s.io/cri-api => k8s.io/cri-api v0.20.0

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.0

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.0

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.0

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.0

replace k8s.io/kubectl => k8s.io/kubectl v0.20.0

replace k8s.io/kubelet => k8s.io/kubelet v0.20.0

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.0

replace k8s.io/metrics => k8s.io/metrics v0.20.0

replace k8s.io/node-api => k8s.io/node-api v0.20.0

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.0

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.20.0

replace k8s.io/sample-controller => k8s.io/sample-controller v0.20.0

replace k8s.io/api => k8s.io/api v0.20.0

replace k8s.io/apimachinery => k8s.io/apimachinery v0.20.0

replace k8s.io/client-go => k8s.io/client-go v0.20.0

replace k8s.io/component-helpers => k8s.io/component-helpers v0.20.0

replace k8s.io/controller-manager => k8s.io/controller-manager v0.20.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.0

replace k8s.io/mount-utils => k8s.io/mount-utils v0.20.0
