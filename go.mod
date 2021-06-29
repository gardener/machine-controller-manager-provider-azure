module github.com/gardener/machine-controller-manager-provider-azure

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v50.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.17
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/gardener/machine-controller-manager v0.39.0
	github.com/golang/mock v1.4.4
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.11.0
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/component-base v0.20.5
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.4.0
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.20.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.6-rc.0
	k8s.io/apiserver => k8s.io/apiserver v0.20.5
	k8s.io/client-go => k8s.io/client-go v0.20.5
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.5
	k8s.io/code-generator => k8s.io/code-generator v0.20.6-rc.0
	k8s.io/kube-openapi => github.com/gardener/kube-openapi v0.0.0-20201221124747-75e88872edcf // k8s-1.19
)
