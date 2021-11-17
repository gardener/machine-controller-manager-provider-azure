module github.com/gardener/machine-controller-manager-provider-azure

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v50.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.17
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/gardener/machine-controller-manager v0.41.0
	github.com/golang/mock v1.4.4
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.11.0
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/component-base v0.20.6
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
)
