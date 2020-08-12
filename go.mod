module github.com/softonic/rate-limit-operator

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	gopkg.in/yaml.v2 v2.3.0
	istio.io/api v0.0.0-20200724154434-34e474846e0d
	istio.io/gogo-genproto v0.0.0-20200709220749-8607e17318e8
	istio.io/pkg v0.0.0-20200709220414-14d5de656564
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/cli-runtime v0.18.0
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
)
