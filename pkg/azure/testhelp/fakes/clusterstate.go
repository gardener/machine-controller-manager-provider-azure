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
)

type ClusterState struct {
	mutex               sync.RWMutex
	ProviderSpec        api.AzureProviderSpec
	MachineResourcesMap map[string]MachineResources
	// currently we support only one vm image as that is sufficient for unit testing.
	VMImageSpec *VMImageSpec
	// currently we support only one agreement terms as that is sufficient for unit testing.
	AgreementTerms *armmarketplaceordering.AgreementTerms
	// currently we only support one subnet as that is sufficient for unit testing.
	SubnetSpec *SubnetSpec
}

type SubnetSpec struct {
	ResourceGroup string // this can be different from the ClusterState.ResourceGroup
	SubnetName    string
	VnetName      string
}

type VMImageSpec struct {
	Publisher string
	Offer     string
	SKU       string
	Version   string
	OfferType armmarketplaceordering.OfferType
}

type DiskType string

const (
	DiskTypeOS       DiskType = "OSDisk"
	DiskTypeData     DiskType = "DataDisk"
	defaultOfferType          = armmarketplaceordering.OfferTypeVirtualmachine
)

func NewClusterState(providerSpec api.AzureProviderSpec) *ClusterState {
	return &ClusterState{
		ProviderSpec:        providerSpec,
		MachineResourcesMap: make(map[string]MachineResources),
	}
}

func (c *ClusterState) AddMachineResources(m MachineResources) {
	c.MachineResourcesMap[m.Name] = m
}

func (c *ClusterState) WithVMImageSpec(vmImageSpec VMImageSpec) *ClusterState {
	c.VMImageSpec = &vmImageSpec
	return c
}

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
		Type: to.Ptr(string(MarketPlaceOrderingOfferType)),
	}
	return c
}

func (c *ClusterState) WithSubnet(resourceGroup, subnetName, vnetName string) *ClusterState {
	c.SubnetSpec = &SubnetSpec{
		ResourceGroup: resourceGroup,
		SubnetName:    subnetName,
		VnetName:      vnetName,
	}
	return c
}

func (c *ClusterState) ResourceGroupExists(resourceGroupName string) bool {
	return c.ProviderSpec.ResourceGroup == resourceGroupName || (c.SubnetSpec != nil && c.SubnetSpec.ResourceGroup == resourceGroupName)
}

func (c *ClusterState) GetVMImage(vmImageSpec VMImageSpec) *armcompute.VirtualMachineImage {
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
			Type: to.Ptr(string(SubnetResourceType)),
		}
	}
	return nil
}

func (c *ClusterState) GetVM(vmName string) *armcompute.VirtualMachine {
	if machineResources, ok := c.MachineResourcesMap[vmName]; ok {
		return machineResources.VM
	}
	return nil
}

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

func (c *ClusterState) CreateVM(resourceGroup string, vmParams armcompute.VirtualMachine) *armcompute.VirtualMachine {
	vmName := *vmParams.Name
	machineResources, ok := c.MachineResourcesMap[vmName]
	if ok {
		newVM := vmParams
		newVM.ID = to.Ptr(CreateVirtualMachineID(testhelp.SubscriptionID, resourceGroup, vmName))
		machineResources.VM = &newVM
		c.MachineResourcesMap[vmName] = machineResources
		return machineResources.VM
	}
	dataDisksConfigured := !utils.IsSliceNilOrEmpty(c.ProviderSpec.Properties.StorageProfile.DataDisks)
	machineResources = NewMachineResourcesBuilder(c.ProviderSpec, vmName).BuildWith(true, false, true, dataDisksConfigured, nil)
	c.MachineResourcesMap[vmName] = machineResources

	return machineResources.VM
}

func (c *ClusterState) GetNIC(nicName string) *armnetwork.Interface {
	for _, m := range c.MachineResourcesMap {
		if m.NIC != nil && *m.NIC.Name == nicName {
			return m.NIC
		}
	}
	return nil
}

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

func (c *ClusterState) ExtractVMNamesFromNICs() []string {
	vmNames := make([]string, 0, len(c.MachineResourcesMap))
	for vmName, mr := range c.MachineResourcesMap {
		if mr.NIC != nil {
			vmNames = append(vmNames, vmName)
		}
	}
	return vmNames
}

func (c *ClusterState) GetAllExistingVMNames() []string {
	vmNames := make([]string, 0, len(c.MachineResourcesMap))
	for vmName, mr := range c.MachineResourcesMap {
		if mr.VM != nil {
			vmNames = append(vmNames, vmName)
		}
	}
	return vmNames
}

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
