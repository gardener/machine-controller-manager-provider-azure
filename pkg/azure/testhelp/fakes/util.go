package fakes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

func CreateProviderSecret() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			api.ClientID:       []byte(testhelp.ClientID),
			api.ClientSecret:   []byte(testhelp.ClientSecret),
			api.SubscriptionID: []byte(testhelp.SubscriptionID),
			api.TenantID:       []byte(testhelp.TenantID),
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

func CreateSubnetName(shootNs string) string {
	return fmt.Sprintf("%s-nodes", shootNs)
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

func CreateAzureDiskPVSource(resourceGroup, diskName string) corev1.PersistentVolumeSource {
	diskURI := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/disks/%s", testhelp.SubscriptionID, resourceGroup, diskName)
	return corev1.PersistentVolumeSource{
		AzureDisk: &corev1.AzureDiskVolumeSource{
			DiskName:    diskName,
			DataDiskURI: diskURI,
			CachingMode: to.Ptr(corev1.AzureDataDiskCachingReadWrite),
			FSType:      to.Ptr("ext4"),
			ReadOnly:    to.Ptr(false),
			Kind:        to.Ptr(corev1.AzureManagedDisk),
		}}
}

func CreateCSIPVSource(driverName, volumeName string) corev1.PersistentVolumeSource {
	return corev1.PersistentVolumeSource{
		CSI: &corev1.CSIPersistentVolumeSource{
			Driver:       driverName,
			VolumeHandle: volumeName,
			ReadOnly:     false,
			FSType:       "ext4",
		}}
}

func GetDefaultVMImageParts() (publisher string, offer string, sku string, version string) {
	urnParts := strings.Split(testhelp.DefaultImageRefURN, ":")
	publisher = urnParts[0]
	offer = urnParts[1]
	sku = urnParts[2]
	version = urnParts[3]
	return
}

func ActualSliceEqualsExpectedSlice[T comparable](actual []T, expected []T) bool {
	actualSet := sets.New[T](actual...)
	expectedSet := sets.New[T](expected...)
	return len(actualSet.Difference(expectedSet)) == 0 && len(expectedSet.Difference(actualSet)) == 0
}

func IsSubnetURIPath(uriPath string, subscriptionID string, subnetSpec SubnetSpec) bool {
	expectedSubnetURIPath := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s", subscriptionID, subnetSpec.ResourceGroup, subnetSpec.VnetName, subnetSpec.SubnetName)
	return uriPath == expectedSubnetURIPath
}

func IsVMImageURIPath(uriPath string, subscriptionID, location string, vmImageSpec VMImageSpec) bool {
	expectedVMImageURIPath := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/publishers/%s/artifacttypes/vmimage/offers/%s/skus/%s/versions/%s", subscriptionID, location, vmImageSpec.Publisher, vmImageSpec.Offer, vmImageSpec.SKU, vmImageSpec.Version)
	return uriPath == expectedVMImageURIPath
}

func IsMktPlaceAgreementURIPath(uriPath string, subscriptionID string, vmImageSpec VMImageSpec) bool {
	expectedAgreementURIPath := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.MarketplaceOrdering/offerTypes/virtualmachine/publishers/%s/offers/%s/plans/%s/agreements/current", subscriptionID, vmImageSpec.Publisher, vmImageSpec.Offer, vmImageSpec.SKU)
	return uriPath == expectedAgreementURIPath
}

func IsNicURIPath(uriPath, subscriptionID, resourceGroup, nicName string) bool {
	expectNicURIPath := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", subscriptionID, resourceGroup, nicName)
	return uriPath == expectNicURIPath
}

func IsVMURIPath(uriPath, subscriptionID, resourceGroup, vmName string) bool {
	expectedVmURIPath := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionID, resourceGroup, vmName)
	return uriPath == expectedVmURIPath
}
