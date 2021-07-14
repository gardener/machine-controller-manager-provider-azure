module github.com/gardener/machine-controller-manager-provider-azure

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v50.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.17
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/gardener/machine-controller-manager v0.40.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.11.0
	github.com/prometheus/client_golang v1.5.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.16.8
	k8s.io/apimachinery v0.16.8
	k8s.io/component-base v0.16.8
	k8s.io/klog v1.0.0
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.0.0-20190918155943-95b840bb6a1f // kubernetes-1.16.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655 // kubernetes-1.16.0
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918160949-bfa5e2e684ad // kubernetes-1.16.0
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90 // kubernetes-1.16.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918163108-da9fdfce26bb // kubernetes-1.16.0
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190912054826-cd179ad6a269 // kubernetes-1.16.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
)
