module github.com/alibaba/kubedl

go 1.16

require (
	github.com/aliyun/aliyun-log-go-sdk v0.1.6
	github.com/fluid-cloudnative/fluid v0.7.0
	github.com/gin-contrib/sessions v0.0.3
	github.com/gin-contrib/static v0.0.0-20191128031702-f81c604d8ac2
	github.com/gin-gonic/gin v1.7.0
	github.com/go-logr/logr v1.2.3
	github.com/go-logr/zapr v0.4.1-0.20210423233217-9f3e0b1ce51b // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/jinzhu/gorm v1.9.12
	github.com/kubernetes-sigs/kube-batch v0.0.0-20200402033359-1ebe60e4af4f
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.24.2
	github.com/openkruise/kruise v1.1.0
	github.com/pierrec/lz4 v2.4.1+incompatible // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.0
	github.com/tidwall/gjson v1.8.0
	golang.org/x/net v0.4.0
	gopkg.in/yaml.v2 v2.4.0
	istio.io/api v0.0.0-20210420211535-1c598ea4139c
	istio.io/client-go v1.9.4
	istio.io/gogo-genproto v0.0.0-20210113155706-4daf5697332f // indirect
	k8s.io/api v0.26.1
	k8s.io/apimachinery v0.26.1
	k8s.io/apiserver v0.22.6
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.22.6
	k8s.io/component-base v0.22.6
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.22.6
	k8s.io/utils v0.0.0-20221107191617-1a15be271d1d
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/controller-tools v0.11.3
	sigs.k8s.io/scheduler-plugins v0.19.9
	sigs.k8s.io/yaml v1.3.0
	volcano.sh/apis v1.3.0-k8s1.18.3-alpha.3
)

replace (
	k8s.io/api => k8s.io/api v0.22.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.6
	k8s.io/apiserver => k8s.io/apiserver v0.22.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.6
	k8s.io/client-go => k8s.io/client-go v0.22.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.6
	k8s.io/code-generator => k8s.io/code-generator v0.22.6
	k8s.io/component-base => k8s.io/component-base v0.22.6
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.6
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.6
	k8s.io/cri-api => k8s.io/cri-api v0.22.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.6
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.6
	k8s.io/kubectl => k8s.io/kubectl v0.22.6
	k8s.io/kubelet => k8s.io/kubelet v0.22.6
	k8s.io/kubernetes => k8s.io/kubernetes v1.22.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.6
	k8s.io/metrics => k8s.io/metrics v0.22.6
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.6
	k8s.io/node-api => k8s.io/node-api v0.22.6
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.22.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.6
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.22.6
	k8s.io/sample-controller => k8s.io/sample-controller v0.22.6
)