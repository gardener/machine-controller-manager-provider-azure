package fakes

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
)

type ClusterState struct {
	mutex               sync.RWMutex
	ResourceGroup       string
	MachineResourcesMap map[string]MachineResources
	vmImageSpec         *VMImageSpec
	agreementTerms      *armmarketplaceordering.AgreementTerms
}

type VMImageSpec struct {
	Publisher string
	Offer     string
	SKU       string
	Version   string
	OfferType string
}

type DiskType string

const (
	DiskTypeOS   DiskType = "OSDisk"
	DiskTypeData DiskType = "DataDisk"
)

func NewClusterState(resourceGroup string) *ClusterState {
	return &ClusterState{
		ResourceGroup:       resourceGroup,
		MachineResourcesMap: make(map[string]MachineResources),
	}
}

func (c *ClusterState) AddMachineResources(m MachineResources) {
	c.MachineResourcesMap[m.Name] = m
}

func (c *ClusterState) WithVMImageSpec(vmImageSpec VMImageSpec) *ClusterState {
	c.vmImageSpec = &vmImageSpec
	return c
}

// WithAgreementTerms initializes ClusterState with a default AgreementTerms resource
// It is mandatory that ClusterState has been initialized with a VMImageSpec. Without a VMImage
// it does not make any sense to have an agreement since there will be no purchase plan.
// It is assumed here that the VMImage is a marketplace image and not a community image.
func (c *ClusterState) WithAgreementTerms(accepted bool) *ClusterState {
	if c.vmImageSpec == nil {
		// do not create any agreement terms
		return c
	}
	// compare relevant fields
	id := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.MarketplaceOrdering/offerTypes/VirtualMachine/publishers/%s/offers/%s/plans/%s/agreements/current", testhelp.SubscriptionID, c.vmImageSpec.Publisher, c.vmImageSpec.Offer, c.vmImageSpec.SKU)
	c.agreementTerms = &armmarketplaceordering.AgreementTerms{
		Properties: &armmarketplaceordering.AgreementProperties{
			Accepted:  to.Ptr(accepted),
			Plan:      to.Ptr(c.vmImageSpec.SKU),
			Product:   to.Ptr(c.vmImageSpec.Offer),
			Publisher: to.Ptr(c.vmImageSpec.Publisher),
		},
		ID:   &id,
		Name: to.Ptr(c.vmImageSpec.SKU),
		Type: to.Ptr(string(MarketPlaceOrderingOfferType)),
	}
	return c
}

func (c *ClusterState) GetVMImage(vmImageSpec VMImageSpec) *armcompute.VirtualMachineImage {
	if c.vmImageSpec == nil || !reflect.DeepEqual(vmImageSpec, c.vmImageSpec) {
		return nil
	}
	id := fmt.Sprintf("/Subscriptions/%s/Providers/Microsoft.Compute/Locations/%s/Publishers/%s/ArtifactTypes/VMImage/Offers/%s/Skus/%s/Versions/%s", testhelp.SubscriptionID, testhelp.Location, c.vmImageSpec.Publisher, c.vmImageSpec.Offer, c.vmImageSpec.SKU, c.vmImageSpec.Version)
	return &armcompute.VirtualMachineImage{
		Location: to.Ptr(testhelp.Location),
		Name:     to.Ptr(c.vmImageSpec.Version),
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
				Name:      to.Ptr(c.vmImageSpec.SKU),
				Product:   to.Ptr(c.vmImageSpec.Offer),
				Publisher: to.Ptr(c.vmImageSpec.Publisher),
			},
		},
	}
}

func (c *ClusterState) GetAgreementTerms(offerType armmarketplaceordering.OfferType, publisherID string, offerID string, planID string) *armmarketplaceordering.AgreementTerms {
	if c.agreementTerms == nil || c.vmImageSpec == nil {
		return nil
	}
	if offerType == armmarketplaceordering.OfferTypeVirtualmachine &&
		publisherID == c.vmImageSpec.Publisher &&
		offerID == c.vmImageSpec.Offer &&
		planID == c.vmImageSpec.SKU {
		return c.agreementTerms
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
