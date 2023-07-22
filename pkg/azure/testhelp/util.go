package testhelp

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func CreateMachineClass(providerSpec api.AzureProviderSpec, resourceGroup *string) (*v1alpha1.MachineClass, error) {
	if resourceGroup != nil {
		providerSpec.ResourceGroup = *resourceGroup
	}
	specBytes, err := json.Marshal(providerSpec)
	if err != nil {
		return nil, err
	}
	machineClass := &v1alpha1.MachineClass{
		Provider: "Azure",
		ProviderSpec: runtime.RawExtension{
			Raw:    specBytes,
			Object: nil,
		},
	}
	return machineClass, nil
}

func NewMachineObjectMeta(namespace string, vmName string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: namespace,
		Name:      vmName,
	}
}

func GetCascadeDeleteOptForNIC(vm armcompute.VirtualMachine) *armcompute.DeleteOptions {
	if vm.Properties != nil && vm.Properties.NetworkProfile != nil && !utils.IsSliceNilOrEmpty(vm.Properties.NetworkProfile.NetworkInterfaces) {
		nic := vm.Properties.NetworkProfile.NetworkInterfaces[0]
		if nic.Properties != nil && nic.Properties.DeleteOption != nil {
			return nic.Properties.DeleteOption
		} else {
			return to.Ptr(armcompute.DeleteOptionsDetach)
		}
	}
	return nil
}

func GetCascadeDeleteOptForOsDisk(vm armcompute.VirtualMachine) *armcompute.DiskDeleteOptionTypes {
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.OSDisk != nil {
		if vm.Properties.StorageProfile.OSDisk.DeleteOption != nil {
			return vm.Properties.StorageProfile.OSDisk.DeleteOption
		} else {
			return to.Ptr(armcompute.DiskDeleteOptionTypesDetach)
		}
	}
	return nil
}

func GetCascadeDeleteOptForDataDisks(vm armcompute.VirtualMachine) map[string]*armcompute.DiskDeleteOptionTypes {
	deleteOpts := make(map[string]*armcompute.DiskDeleteOptionTypes)
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && !utils.IsSliceNilOrEmpty(vm.Properties.StorageProfile.DataDisks) {
		existingDataDisks := vm.Properties.StorageProfile.DataDisks
		for _, dataDisk := range existingDataDisks {
			deleteOpt := to.Ptr(armcompute.DiskDeleteOptionTypesDetach)
			if dataDisk.DeleteOption != nil {
				deleteOpt = dataDisk.DeleteOption
			}
			deleteOpts[*dataDisk.Name] = deleteOpt
		}
	}
	return deleteOpts
}
