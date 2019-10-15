module github.com/kubernetes-csi/external-attacher

go 1.12

require (
	github.com/container-storage-interface/spec v1.1.0
	github.com/davecgh/go-spew v1.1.1
	github.com/evanphx/json-patch v4.2.0+incompatible
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.3.1
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/kubernetes-csi/csi-lib-utils v0.6.1
	github.com/kubernetes-csi/csi-test v2.0.0+incompatible
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/genproto v0.0.0-20190418145605-e7d98fc518a7 // indirect
	google.golang.org/grpc v1.19.1
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20191005115622-2e41325d9e4b
	k8s.io/apimachinery v0.0.0-20191006235458-f9f2f3f8ab02
	k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
	k8s.io/csi-translation-lib v0.0.0-20191009030015-17db17aaadeb
	k8s.io/klog v1.0.0
)

replace k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918201827-3de75813f604

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d

replace k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918200908-1e17798da8c1

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918202139-0b14c719ca62

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918203125-ae665f80358a

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918202959-c340507a5d48

replace k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b

replace k8s.io/component-base => k8s.io/component-base v0.0.0-20190918200425-ed2f0867c778

replace k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918201136-c3a845f1fbb2

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918202837-c54ce30c680e

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918202429-08c8357f8e2d

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918202713-c34a54b3ec8e

replace k8s.io/kubectl => k8s.io/kubectl v0.0.0-20190602132728-7075c07e78bf

replace k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918202550-958285cf3eef

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918203421-225f0541b3ea

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20190918202012-3c1ca76f5bda

replace k8s.io/node-api => k8s.io/node-api v0.0.0-20190918203548-2c4c2679bece

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918201353-5cc279503896

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20190918202305-ed68a9f09ae1

replace k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20190918201537-fabef0de90df
