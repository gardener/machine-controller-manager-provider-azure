// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// CreateProviderSecret creates a fake secret containing provider credentials.
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

// CreateVirtualMachineID creates an azure representation of virtual machine ID.
func CreateVirtualMachineID(subscriptionID, resourceGroup, vmName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionID, resourceGroup, vmName)
}

// CreateNetworkInterfaceID creates an azure representation of network ID.
func CreateNetworkInterfaceID(subscriptionID, resourceGroup, nicName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", subscriptionID, resourceGroup, nicName)
}

// CreateIPConfigurationID creates an azure representation of IP configuration ID.
func CreateIPConfigurationID(subscriptionID, resourceGroup, nicName, ipConfigName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s/ipConfigurations/%s", subscriptionID, resourceGroup, nicName, ipConfigName)
}

// CreateSubnetName creates a subnet name.
func CreateSubnetName(shootNs string) string {
	return fmt.Sprintf("%s-nodes", shootNs)
}

// CreateMachineClass creates a v1alpha1.MachineClass from resource group and provider spec.
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

// NewMachineObjectMeta creates an ObjectMeta for a Machine.
func NewMachineObjectMeta(namespace string, vmName string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: namespace,
		Name:      vmName,
	}
}

// GetCascadeDeleteOptForNIC gets the configured delete option for a NIC associated to the passed VM.
func GetCascadeDeleteOptForNIC(vm armcompute.VirtualMachine) *armcompute.DeleteOptions {
	if vm.Properties != nil && vm.Properties.NetworkProfile != nil && !utils.IsSliceNilOrEmpty(vm.Properties.NetworkProfile.NetworkInterfaces) {
		nic := vm.Properties.NetworkProfile.NetworkInterfaces[0]
		if nic.Properties != nil && nic.Properties.DeleteOption != nil {
			return nic.Properties.DeleteOption
		}
		return to.Ptr(armcompute.DeleteOptionsDetach)
	}
	return nil
}

// GetCascadeDeleteOptForOsDisk gets the configured delete option for OSDisk associated to the passed VM.
func GetCascadeDeleteOptForOsDisk(vm armcompute.VirtualMachine) *armcompute.DiskDeleteOptionTypes {
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.OSDisk != nil {
		if vm.Properties.StorageProfile.OSDisk.DeleteOption != nil {
			return vm.Properties.StorageProfile.OSDisk.DeleteOption
		}
		return to.Ptr(armcompute.DiskDeleteOptionTypesDetach)
	}
	return nil
}

// GetCascadeDeleteOptForDataDisks gets the configured delete option for DataDisk associated to the passed VM.
// Returns a map whose key is the data disk name and the value is the delete option.
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

// CreateAzureDiskPVSource creates a corev1.PersistentVolumeSource initializing AzureDisk.
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

// CreateCSIPVSource creates a corev1.PersistentVolumeSource initializing CSI.
func CreateCSIPVSource(driverName, volumeName string) corev1.PersistentVolumeSource {
	return corev1.PersistentVolumeSource{
		CSI: &corev1.CSIPersistentVolumeSource{
			Driver:       driverName,
			VolumeHandle: volumeName,
			ReadOnly:     false,
			FSType:       "ext4",
		}}
}

// GetDefaultVMImageParts splits testhelp.DefaultImageRefURN into its constituent parts.
func GetDefaultVMImageParts() (publisher string, offer string, sku string, version string) {
	urnParts := strings.Split(testhelp.DefaultImageRefURN, ":")
	publisher = urnParts[0]
	offer = urnParts[1]
	sku = urnParts[2]
	version = urnParts[3]
	return
}

// ActualSliceEqualsExpectedSlice creates a utility comparator comparing two slices for equality.
func ActualSliceEqualsExpectedSlice[T comparable](actual []T, expected []T) bool {
	actualSet := sets.New[T](actual...)
	expectedSet := sets.New[T](expected...)
	return len(actualSet.Difference(expectedSet)) == 0 && len(expectedSet.Difference(actualSet)) == 0
}

// IsSubnetURIPath checks if the URI points to a subnet resource.
func IsSubnetURIPath(uriPath string, subscriptionID string, subnetSpec SubnetSpec) bool {
	expectedSubnetURIPath := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s", subscriptionID, subnetSpec.ResourceGroup, subnetSpec.VnetName, subnetSpec.SubnetName)
	return uriPath == expectedSubnetURIPath
}

// IsVMImageURIPath checks if the URI points to a VM Image resource.
func IsVMImageURIPath(uriPath string, subscriptionID, location string, vmImageSpec VMImageSpec) bool {
	expectedVMImageURIPath := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/publishers/%s/artifacttypes/vmimage/offers/%s/skus/%s/versions/%s", subscriptionID, location, vmImageSpec.Publisher, vmImageSpec.Offer, vmImageSpec.SKU, vmImageSpec.Version)
	return uriPath == expectedVMImageURIPath
}

// IsMktPlaceAgreementURIPath checks if the URI points to a market-place agreement resource.
func IsMktPlaceAgreementURIPath(uriPath string, subscriptionID string, vmImageSpec VMImageSpec) bool {
	expectedAgreementURIPath := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.MarketplaceOrdering/offerTypes/virtualmachine/publishers/%s/offers/%s/plans/%s/agreements/current", subscriptionID, vmImageSpec.Publisher, vmImageSpec.Offer, vmImageSpec.SKU)
	return uriPath == expectedAgreementURIPath
}

// IsNicURIPath checks if the URI points to a NIC resource.
func IsNicURIPath(uriPath, subscriptionID, resourceGroup, nicName string) bool {
	expectNicURIPath := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", subscriptionID, resourceGroup, nicName)
	return uriPath == expectNicURIPath
}

// IsVMURIPath checks if the URI points to a VM resource.
func IsVMURIPath(uriPath, subscriptionID, resourceGroup, vmName string) bool {
	expectedVMURIPath := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionID, resourceGroup, vmName)
	return uriPath == expectedVMURIPath
}
