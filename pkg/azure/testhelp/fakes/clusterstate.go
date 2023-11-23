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
	"fmt"
	"reflect"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"

	"golang.org/x/exp/slices"
)

// ClusterState is a holder of state of cluster resources.
// During unit testing all fake client implementations will operate on this shared cluster state thus allowing
// all clients to see the same state. To prevent more than one client to mutate the state a mutex has to be acquired
// before making modifications.
// NOTE: It is recommended that tests do not share a ClusterState.
type ClusterState struct {
	mutex sync.RWMutex
	// ProviderSpec is the azure provider spec to be used by all clients.
	ProviderSpec api.AzureProviderSpec
	// MachineResourcesMap is a map where key is the name of the VM which is also the name for the machine.
	// The value is a MachineResources object.
	MachineResourcesMap map[string]MachineResources
	// VMImageSpec is the VM image spec for this cluster state.
	// Currently, we support only one vm image as that is sufficient for unit testing.
	VMImageSpec *VMImageSpec
	// AgreementTerms is the agreement terms for the VM Image.
	// Currently, we support only one agreement terms as that is sufficient for unit testing.
	AgreementTerms *armmarketplaceordering.AgreementTerms
	// SubnetSpec is the subnet spec that is used to configure all NICs.
	// Currently, we only support one subnet as that is sufficient for unit testing.
	SubnetSpec *SubnetSpec
}

// SubnetSpec is the spec that captures the subnet configuration.
type SubnetSpec struct {
	// ResourceGroup is the resource group where the subnet is defined.
	ResourceGroup string
	// SubnetName is the name of the subnet.
	SubnetName string
	// VnetName is the name of the virtual network.
	VnetName string
}

// VMImageSpec is the spec for the VM Image.
type VMImageSpec struct {
	// Publisher is the publisher ID of the image.
	Publisher string
	// Offer of the platform/market-place image used to create the VM.
	Offer string
	// SKU is an instance of an offer, such as a major release of a distribution.
	SKU string
	// Version is the version number of an image SKU.
	Version string
	// OfferType is the offer type. E.g. armmarketplaceordering.OfferTypeVirtualmachine.
	OfferType armmarketplaceordering.OfferType
}

// DiskType is used as an enum type to define types of disks that can be associated to a VM.
type DiskType string

const (
	// DiskTypeOS is the OS Disk that is always associated to a VM.
	DiskTypeOS DiskType = "OSDisk"
	// DiskTypeData represents one or more data disks that can be associated to a VM.
	DiskTypeData     DiskType = "DataDisk"
	defaultOfferType          = armmarketplaceordering.OfferTypeVirtualmachine
)

// NewClusterState creates a new ClusterState.
func NewClusterState(providerSpec api.AzureProviderSpec) *ClusterState {
	return &ClusterState{
		ProviderSpec:        providerSpec,
		MachineResourcesMap: make(map[string]MachineResources),
	}
}

// AddMachineResources adds MachineResources against a VM/Machine name.
func (c *ClusterState) AddMachineResources(m MachineResources) {
	c.MachineResourcesMap[m.Name] = m
}

// ClusterState builder methods
// ----------------------------------------------------------------------------------------------------------

// WithVMImageSpec initializes ClusterState with the passed in VM Image spec and returns the ClusterState.
func (c *ClusterState) WithVMImageSpec(vmImageSpec VMImageSpec) *ClusterState {
	c.VMImageSpec = &vmImageSpec
	return c
}

// WithDefaultVMImageSpec initializes ClusterState with a default VMImage and returns the ClusterState.
func (c *ClusterState) WithDefaultVMImageSpec() *ClusterState {
	publisher, offer, sku, version := GetDefaultVMImageParts()
	c.VMImageSpec = &VMImageSpec{
		Publisher: publisher,
		Offer:     offer,
		SKU:       sku,
		Version:   version,
		OfferType: defaultOfferType,
	}
	return c
}

// WithAgreementTerms initializes ClusterState with a default AgreementTerms resource
// It is mandatory that ClusterState has been initialized with a VMImageSpec. Without a VMImage
// it does not make any sense to have an agreement since there will be no purchase plan.
// It is assumed here that the VMImage is a marketplace image and not a community image.
func (c *ClusterState) WithAgreementTerms(accepted bool) *ClusterState {
	if c.VMImageSpec == nil {
		// do not create any agreement terms
		return c
	}
	// compare relevant fields
	id := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.MarketplaceOrdering/offerTypes/VirtualMachine/publishers/%s/offers/%s/plans/%s/agreements/current", testhelp.SubscriptionID, c.VMImageSpec.Publisher, c.VMImageSpec.Offer, c.VMImageSpec.SKU)
	c.AgreementTerms = &armmarketplaceordering.AgreementTerms{
		Properties: &armmarketplaceordering.AgreementProperties{
			Accepted:  to.Ptr(accepted),
			Plan:      to.Ptr(c.VMImageSpec.SKU),
			Product:   to.Ptr(c.VMImageSpec.Offer),
			Publisher: to.Ptr(c.VMImageSpec.Publisher),
		},
		ID:   &id,
		Name: to.Ptr(c.VMImageSpec.SKU),
		Type: to.Ptr(string(utils.MarketPlaceOrderingOfferType)),
	}
	return c
}

// WithSubnet initializes ClusterState with subnet and returns the ClusterState.
func (c *ClusterState) WithSubnet(resourceGroup, subnetName, vnetName string) *ClusterState {
	c.SubnetSpec = &SubnetSpec{
		ResourceGroup: resourceGroup,
		SubnetName:    subnetName,
		VnetName:      vnetName,
	}
	return c
}

// ----------------------------------------------------------------------------------------------------------

// ResourceGroupExists checks if a passed in resourceGroupName has been configured in the ClusterState.
func (c *ClusterState) ResourceGroupExists(resourceGroupName string) bool {
	return c.ProviderSpec.ResourceGroup == resourceGroupName || (c.SubnetSpec != nil && c.SubnetSpec.ResourceGroup == resourceGroupName)
}

// GetVirtualMachineImage gets an armcompute.VirtualMachineImage from a VMImageSpec.
func (c *ClusterState) GetVirtualMachineImage(vmImageSpec VMImageSpec) *armcompute.VirtualMachineImage {
	if c.VMImageSpec == nil || !reflect.DeepEqual(vmImageSpec, *c.VMImageSpec) {
		return nil
	}
	id := fmt.Sprintf("/Subscriptions/%s/Providers/Microsoft.Compute/Locations/%s/Publishers/%s/ArtifactTypes/VMImage/Offers/%s/Skus/%s/Versions/%s", testhelp.SubscriptionID, testhelp.Location, c.VMImageSpec.Publisher, c.VMImageSpec.Offer, c.VMImageSpec.SKU, c.VMImageSpec.Version)
	return &armcompute.VirtualMachineImage{
		Location: to.Ptr(testhelp.Location),
		Name:     to.Ptr(c.VMImageSpec.Version),
		ID:       &id,
		Properties: &armcompute.VirtualMachineImageProperties{
			Architecture:                 to.Ptr(armcompute.ArchitectureTypesX64),
			AutomaticOSUpgradeProperties: &armcompute.AutomaticOSUpgradeProperties{AutomaticOSUpgradeSupported: to.Ptr(false)},
			Features: []*armcompute.VirtualMachineImageFeature{
				{
					Name:  to.Ptr("IsAcceleratedNetworkSupported"),
					Value: to.Ptr("False"),
				},
				{
					Name:  to.Ptr("DiskControllerTypes"),
					Value: to.Ptr("SCSI"),
				},
				{
					Name:  to.Ptr("IsHibernateSupported"),
					Value: to.Ptr("False"),
				},
			},
			ImageDeprecationStatus: &armcompute.ImageDeprecationStatus{ImageState: to.Ptr(armcompute.ImageStateActive)},
			OSDiskImage:            &armcompute.OSDiskImage{OperatingSystem: to.Ptr(armcompute.OperatingSystemTypesLinux)},
			Plan: &armcompute.PurchasePlan{
				Name:      to.Ptr(c.VMImageSpec.SKU),
				Product:   to.Ptr(c.VMImageSpec.Offer),
				Publisher: to.Ptr(c.VMImageSpec.Publisher),
			},
		},
	}
}

// GetAgreementTerms returns an armmarketplaceordering.AgreementTerms matching passed in offerType, publisherID and offerID.
func (c *ClusterState) GetAgreementTerms(offerType armmarketplaceordering.OfferType, publisherID string, offerID string) *armmarketplaceordering.AgreementTerms {
	if c.AgreementTerms == nil || c.VMImageSpec == nil {
		return nil
	}
	if offerType == armmarketplaceordering.OfferTypeVirtualmachine &&
		publisherID == c.VMImageSpec.Publisher &&
		offerID == c.VMImageSpec.Offer {
		return c.AgreementTerms
	}
	return nil
}

// GetSubnet gets an armnetwork.Subnet if it matches the configured subnet having the same resourceGroup, subnet and vnet names.
func (c *ClusterState) GetSubnet(resourceGroup, subnetName, vnetName string) *armnetwork.Subnet {
	if c.SubnetSpec != nil &&
		c.SubnetSpec.ResourceGroup == resourceGroup &&
		c.SubnetSpec.SubnetName == subnetName &&
		c.SubnetSpec.VnetName == vnetName {
		id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s", testhelp.SubscriptionID, resourceGroup, vnetName, subnetName)
		return &armnetwork.Subnet{
			ID:   to.Ptr(id),
			Name: to.Ptr(subnetName),
			Properties: &armnetwork.SubnetPropertiesFormat{
				PrivateEndpointNetworkPolicies:    to.Ptr(armnetwork.VirtualNetworkPrivateEndpointNetworkPoliciesEnabled),
				PrivateLinkServiceNetworkPolicies: to.Ptr(armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesEnabled),
				ProvisioningState:                 to.Ptr(armnetwork.ProvisioningStateSucceeded),
			},
			Type: to.Ptr(string(utils.SubnetResourceType)),
		}
	}
	return nil
}

// GetVM returns an armcompute.VirtualMachine having the same name as the passed in vmName.
func (c *ClusterState) GetVM(vmName string) *armcompute.VirtualMachine {
	if machineResources, ok := c.MachineResourcesMap[vmName]; ok {
		return machineResources.VM
	}
	return nil
}

// DeleteVM deletes the VM having the same name as passed in vmName from the ClusterState.
func (c *ClusterState) DeleteVM(vmName string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	m, ok := c.MachineResourcesMap[vmName]
	if !ok {
		return
	}
	if m.ShouldCascadeDeleteAllAttachedResources() {
		delete(c.MachineResourcesMap, vmName)
		return
	}

	m.HandleNICOnVMDelete()
	m.HandleOSDiskOnVMDelete()
	m.HandleDataDisksOnVMDelete()
	m.VM = nil

	if !m.HasResources() {
		delete(c.MachineResourcesMap, vmName)
	} else {
		c.MachineResourcesMap[vmName] = m
	}
}

// CreateVM creates a new VM in the resourceGroup using vmParams.
// This new VM will be added to the ClusterState and also returned for consumption.
func (c *ClusterState) CreateVM(resourceGroup string, vmParams armcompute.VirtualMachine) (*armcompute.VirtualMachine, error) {
	vmName := *vmParams.Name
	machineResources, ok := c.MachineResourcesMap[vmName]
	// It is assumed that this method will be called after the NIC referenced in vmParams has been created.
	if ok {
		updateMachineResourcesFromVMParams(c.ProviderSpec, resourceGroup, vmParams, &machineResources)
		c.MachineResourcesMap[vmName] = machineResources
		return machineResources.VM, nil
	}
	referencedNICID := getReferencedNICIDFromVirtualMachine(vmParams)
	var err error
	if referencedNICID != nil {
		err = testhelp.ConfiguredRelatedResourceNotFound(testhelp.ErrorCodeReferencedResourceNotFound, *referencedNICID)
	}
	return nil, err
}

// GetNIC gets a NIC matching the passed name if one exists.
func (c *ClusterState) GetNIC(nicName string) *armnetwork.Interface {
	for _, m := range c.MachineResourcesMap {
		if m.NIC != nil && *m.NIC.Name == nicName {
			return m.NIC
		}
	}
	return nil
}

// DeleteNIC deletes the NIC with the matching nicName.
func (c *ClusterState) DeleteNIC(nicName string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var targetMachineResources *MachineResources
loop:
	for _, m := range c.MachineResourcesMap {
		if m.NIC != nil && *m.NIC.Name == nicName {
			targetMachineResources = &m
			break loop
		}
	}
	if targetMachineResources != nil {
		targetMachineResources.NIC = nil
		if !targetMachineResources.HasResources() {
			delete(c.MachineResourcesMap, targetMachineResources.Name)
		}
		c.MachineResourcesMap[targetMachineResources.Name] = *targetMachineResources
	}
}

// CreateNIC creates a nic with the passed in nicName and nic parameters.
// The nic is also associated with the VM and ClusterState is updated.
func (c *ClusterState) CreateNIC(nicName string, nic *armnetwork.Interface) *armnetwork.Interface {
	vmName := utils.ExtractVMNameFromNICName(nicName)
	machineResources, ok := c.MachineResourcesMap[vmName]
	if !ok {
		machineResources = MachineResources{}
	}
	nicID := CreateNetworkInterfaceID(testhelp.SubscriptionID, c.ProviderSpec.ResourceGroup, nicName)
	machineResources.NIC = nic
	machineResources.NIC.ID = &nicID
	c.MachineResourcesMap[vmName] = machineResources
	return machineResources.NIC
}

// GetDisk gets the Disk matching diskName.
func (c *ClusterState) GetDisk(diskName string) *armcompute.Disk {
	diskType, machine := c.getDiskTypeAndOwningMachineResources(diskName)
	switch diskType {
	case DiskTypeOS:
		return machine.OSDisk
	case DiskTypeData:
		return machine.DataDisks[diskName]
	default:
		return nil
	}
}

// DeleteDisk deletes the disk matching diskName.
func (c *ClusterState) DeleteDisk(diskName string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	diskType, machineResources := c.getDiskTypeAndOwningMachineResources(diskName)
	if machineResources == nil {
		return
	}
	switch diskType {
	case DiskTypeOS:
		machineResources.OSDisk = nil
	case DiskTypeData:
		delete(machineResources.DataDisks, diskName)
		if len(machineResources.DataDisks) == 0 {
			machineResources.DataDisks = nil
		}
	}
	if !machineResources.HasResources() {
		delete(c.MachineResourcesMap, machineResources.Name)
	} else {
		c.MachineResourcesMap[machineResources.Name] = *machineResources
	}
}

// ExtractVMNamesFromNICsMatchingTagKeys extracts VM names from all configured NICs in the ClusterState that has all tagKeys.
func (c *ClusterState) ExtractVMNamesFromNICsMatchingTagKeys(tagKeys []string) []string {
	vmNames := make([]string, 0, len(c.MachineResourcesMap))
	for vmName, mr := range c.MachineResourcesMap {
		if mr.NIC != nil {
			// check if all tag keys are present for this NIC
			if containsAllTagKeys(mr.NIC.Tags, tagKeys) {
				vmNames = append(vmNames, vmName)
			}
		}
	}
	return vmNames
}

// GetVMsMatchingTagKeys returns VM names for all configured VMs in the ClusterState that has all tagKeys.
func (c *ClusterState) GetVMsMatchingTagKeys(tagKeys []string) []string {
	vmNames := make([]string, 0, len(c.MachineResourcesMap))
	for vmName, mr := range c.MachineResourcesMap {
		if mr.VM != nil {
			// check if all tag keys are present for this VM
			if containsAllTagKeys(mr.VM.Tags, tagKeys) {
				vmNames = append(vmNames, vmName)
			}
		}
	}
	return vmNames
}

func containsAllTagKeys(resourceTags map[string]*string, tagKeys []string) bool {
	numMatches := 0
	for k := range resourceTags {
		if slices.Contains(tagKeys, k) {
			numMatches++
		}
	}
	return numMatches == len(tagKeys)
}

// GetAllVMNamesFromMachineResources gets all VM names from all existing MachineResources
// irrespective of if the VM in each MachineResources object exists. This covers the case where
// left over NIC/Disk(s) are there but the corresponding VM is not present.
func (c *ClusterState) GetAllVMNamesFromMachineResources() []string {
	vmNames := make([]string, 0, len(c.MachineResourcesMap))
	for vmName := range c.MachineResourcesMap {
		vmNames = append(vmNames, vmName)
	}
	return vmNames
}

func (c *ClusterState) getDiskTypeAndOwningMachineResources(diskName string) (DiskType, *MachineResources) {
	if c.MachineResourcesMap != nil {
		for _, m := range c.MachineResourcesMap {
			if m.OSDisk != nil && *m.OSDisk.Name == diskName {
				return DiskTypeOS, &m
			}
			if m.DataDisks != nil {
				if _, ok := m.DataDisks[diskName]; ok {
					return DiskTypeData, &m
				}
			}
		}
	}
	return "", nil
}

func getReferencedNICIDFromVirtualMachine(vmParams armcompute.VirtualMachine) *string {
	if vmParams.Properties != nil &&
		vmParams.Properties.NetworkProfile != nil &&
		vmParams.Properties.NetworkProfile.NetworkInterfaces != nil && len(vmParams.Properties.NetworkProfile.NetworkInterfaceConfigurations) > 0 {
		return vmParams.Properties.NetworkProfile.NetworkInterfaces[0].ID
	}
	return nil
}
