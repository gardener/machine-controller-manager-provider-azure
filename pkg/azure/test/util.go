package test

import (
	"fmt"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateProviderSecret() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			api.ClientID:       []byte(ClientID),
			api.ClientSecret:   []byte(ClientSecret),
			api.SubscriptionID: []byte(SubscriptionID),
			api.TenantID:       []byte(TenantID),
		},
	}
}

func CreateVirtualMachineID(subscriptionID, resourceGroup, vmName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionID, resourceGroup, vmName)
}

func CreateNetworkInterfaceID(subscriptionID, resourceGroup, nicName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", subscriptionID, resourceGroup, nicName)
}

func CreateIPConfigurationID(subscriptionID, resourceGroup, nicName, ipConfigName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s/ipConfigurations/%s", subscriptionID, resourceGroup, nicName, ipConfigName)
}
