package fakes

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"k8s.io/utils/pointer"
)

var (
	// CascadeDeleteAllResources creates an instance of CascadeDeleteOpts setting
	// cascade delete for NIC, OSDisk and DataDisks
	CascadeDeleteAllResources = CascadeDeleteOpts{
		NIC:      to.Ptr(armcompute.DeleteOptionsDelete),
		OSDisk:   to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
		DataDisk: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
	}
)

// MachineResources holds VM and all associated resources that are created for the VM.
// This will be used in creating and maintaining fake ClusterState which will be further
// used for unit testing.
type MachineResources struct {
	// Name is the same as the *VM.Name. It is kept separately here to allow MachineResources
	// to be retrieved or deleted completely when the VM does not exist but other resources are left behind.
	Name              string
	VM                *armcompute.VirtualMachine
	OSDisk            *armcompute.Disk
	DataDisks         map[string]*armcompute.Disk
	NIC               *armnetwork.Interface
	cascadeDeleteOpts *CascadeDeleteOpts
}

type CascadeDeleteOpts struct {
	NIC      *armcompute.DeleteOptions
	OSDisk   *armcompute.DiskDeleteOptionTypes
	DataDisk *armcompute.DiskDeleteOptionTypes
}

func (m *MachineResources) ShouldCascadeDeleteAllAttachedResources() bool {
	if m.cascadeDeleteOpts != nil {
		if m.cascadeDeleteOpts.NIC != nil && *m.cascadeDeleteOpts.NIC == armcompute.DeleteOptionsDelete &&
			m.cascadeDeleteOpts.OSDisk != nil && *m.cascadeDeleteOpts.OSDisk == armcompute.DiskDeleteOptionTypesDelete &&
			m.cascadeDeleteOpts.DataDisk != nil && *m.cascadeDeleteOpts.DataDisk == armcompute.DiskDeleteOptionTypesDelete {
			return true
		}
	}
	return false
}

func (m *MachineResources) HandleNICOnMachineDelete() {
	if m.cascadeDeleteOpts != nil && m.cascadeDeleteOpts.NIC != nil && *m.cascadeDeleteOpts.NIC == armcompute.DeleteOptionsDelete {
		m.NIC = nil
	} else {
		m.NIC.Properties.VirtualMachine = nil
	}
}

func (m *MachineResources) HandleOSDiskOnMachineDelete() {
	if m.cascadeDeleteOpts != nil && m.cascadeDeleteOpts.OSDisk != nil && *m.cascadeDeleteOpts.OSDisk == armcompute.DiskDeleteOptionTypesDelete {
		m.OSDisk = nil
	} else {
		m.OSDisk.ManagedBy = nil
	}
}

func (m *MachineResources) HandleDataDisksOnMachineDelete() {
	if m.cascadeDeleteOpts != nil && m.cascadeDeleteOpts.DataDisk != nil && *m.cascadeDeleteOpts.DataDisk == armcompute.DiskDeleteOptionTypesDelete {
		m.DataDisks = nil
	} else {
		if m.DataDisks != nil {
			for _, d := range m.DataDisks {
				d.ManagedBy = nil
			}
		}
	}
}

func (m *MachineResources) HasResources() bool {
	return m.VM != nil || m.NIC != nil || m.OSDisk != nil || (m.DataDisks != nil && len(m.DataDisks) > 0)
}

//----------------------------------------------------------------------
// Builder for MachineResources
//----------------------------------------------------------------------

// MachineResourcesBuilder is a builder for MachineResources
type MachineResourcesBuilder struct {
	spec              api.AzureProviderSpec
	vmName            string
	plan              *armcompute.Plan
	cascadeDeleteOpts *CascadeDeleteOpts
}

// NewMachineResourcesBuilder creates a new instance of MachineResourcesBuilder
func NewMachineResourcesBuilder(spec api.AzureProviderSpec, vmName string) *MachineResourcesBuilder {
	defaultPlan := &armcompute.Plan{
		Name:      to.Ptr("greatest"),
		Product:   to.Ptr("gardenlinux"),
		Publisher: to.Ptr("sap"),
	}
	return &MachineResourcesBuilder{
		spec:   spec,
		vmName: vmName,
		plan:   defaultPlan,
	}
}

func (b *MachineResourcesBuilder) WithPlan(plan armcompute.Plan) *MachineResourcesBuilder {
	b.plan = &plan
	return b
}

func (b *MachineResourcesBuilder) WithCascadeDeleteOptions(opts CascadeDeleteOpts) *MachineResourcesBuilder {
	b.cascadeDeleteOpts = &opts
	return b
}

func (b *MachineResourcesBuilder) BuildAllResources() MachineResources {
	return b.BuildWith(true, true, true, true, nil)
}

func (b *MachineResourcesBuilder) BuildWith(createVM, createNIC, createOSDisk, createDataDisk bool, withNonExistentVMID *string) MachineResources {
	return b.CreateMachineResources(createVM, createNIC, createOSDisk, createDataDisk, withNonExistentVMID)
}

func (b *MachineResourcesBuilder) CreateMachineResources(createVM, createNIC, createOSDisk, createDataDisks bool, nonExistentVMID *string) MachineResources {
	var (
		vm        *armcompute.VirtualMachine
		vmID      = nonExistentVMID
		osDisk    *armcompute.Disk
		dataDisks map[string]*armcompute.Disk
		nic       *armnetwork.Interface
	)
	if createVM {
		vm = createVMResource(b.spec, b.vmName, b.plan, b.cascadeDeleteOpts)
		vmID = vm.ID
	}
	if createNIC {
		nic = CreateNICResource(b.spec, vmID, utils.CreateNICName(b.vmName))
	}
	if createOSDisk {
		osDisk = CreateDiskResource(b.spec, utils.CreateOSDiskName(b.vmName), vmID, b.plan)
	}
	if createDataDisks {
		specDataDisks := b.spec.Properties.StorageProfile.DataDisks
		if specDataDisks != nil {
			dataDisks = make(map[string]*armcompute.Disk, len(specDataDisks))
			for _, specDataDisk := range specDataDisks {
				diskName := utils.CreateDataDiskName(b.vmName, specDataDisk)
				dataDisks[diskName] = CreateDiskResource(b.spec, diskName, vmID, nil)
			}
		}
	}
	return MachineResources{
		Name:              b.vmName,
		VM:                vm,
		OSDisk:            osDisk,
		DataDisks:         dataDisks,
		NIC:               nic,
		cascadeDeleteOpts: b.cascadeDeleteOpts,
	}
}

func CreateNICResource(spec api.AzureProviderSpec, vmID *string, nicName string) *armnetwork.Interface {
	ipConfigID := test.CreateIPConfigurationID(test.SubscriptionID, spec.ResourceGroup, nicName, nicName)
	interfaceID := test.CreateNetworkInterfaceID(test.SubscriptionID, spec.ResourceGroup, nicName)

	return &armnetwork.Interface{
		Location: &spec.Location,
		Properties: &armnetwork.InterfacePropertiesFormat{
			EnableAcceleratedNetworking: spec.Properties.NetworkProfile.AcceleratedNetworking,
			EnableIPForwarding:          to.Ptr(true),
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					ID:         &ipConfigID,
					Name:       to.Ptr(nicName),
					Properties: nil,
					Type:       to.Ptr("Microsoft.Network/networkInterfaces/ipConfigurations"),
				},
			},
			NicType:        to.Ptr(armnetwork.NetworkInterfaceNicTypeStandard),
			VirtualMachine: &armnetwork.SubResource{ID: vmID},
		},
		Tags: createResourceTags(spec.Tags),
		ID:   &interfaceID,
		Name: &nicName,
		Type: to.Ptr("Microsoft.Network/networkInterfaces"),
	}
}

func createVMResource(spec api.AzureProviderSpec, vmName string, plan *armcompute.Plan, cascadeDeleteOpts *CascadeDeleteOpts) *armcompute.VirtualMachine {
	id := test.CreateVirtualMachineID(test.SubscriptionID, spec.ResourceGroup, vmName)
	return &armcompute.VirtualMachine{
		Location: to.Ptr(spec.Location),
		Plan:     plan,
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypes(spec.Properties.HardwareProfile.VMSize)),
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: to.Ptr(utils.CreateNICName(vmName)),
						Properties: &armcompute.NetworkInterfaceReferenceProperties{
							DeleteOption: cascadeDeleteOpts.NIC,
							Primary:      to.Ptr(true),
						},
					},
				},
			},
			OSProfile: &armcompute.OSProfile{
				AdminUsername: to.Ptr(spec.Properties.OsProfile.AdminUsername),
			},
			StorageProfile: &armcompute.StorageProfile{
				DataDisks:      createDataDisks(spec, vmName, cascadeDeleteOpts.DataDisk),
				ImageReference: createImageReference(spec.Properties.StorageProfile.ImageReference),
				OSDisk: &armcompute.OSDisk{
					CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesEmpty),
					Caching:      to.Ptr(armcompute.CachingTypes(spec.Properties.StorageProfile.OsDisk.Caching)),
					DeleteOption: cascadeDeleteOpts.OSDisk,
					DiskSizeGB:   pointer.Int32(spec.Properties.StorageProfile.OsDisk.DiskSizeGB),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						StorageAccountType: to.Ptr(armcompute.StorageAccountTypes(spec.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType)),
					},
					Name:   to.Ptr(utils.CreateOSDiskName(vmName)),
					OSType: to.Ptr(armcompute.OperatingSystemTypesLinux),
				},
			},
		},
		Tags:  createResourceTags(spec.Tags),
		Zones: []*string{to.Ptr("1")},
		Name:  to.Ptr(vmName),
		ID:    to.Ptr(id),
		Type:  to.Ptr("Microsoft.Compute/virtualMachines"),
	}
}

func createImageReference(imageRef api.AzureImageReference) *armcompute.ImageReference {
	var (
		id        *string
		publisher *string
		sku       *string
		offer     *string
		version   *string
	)
	if !utils.IsEmptyString(imageRef.ID) {
		id = to.Ptr(imageRef.ID)
	}
	if !utils.IsNilOrEmptyStringPtr(imageRef.URN) {
		urnParts := strings.Split(*imageRef.URN, ":")
		publisher = to.Ptr(urnParts[0])
		offer = to.Ptr(urnParts[1])
		sku = to.Ptr(urnParts[2])
		version = to.Ptr(urnParts[3])
	}
	return &armcompute.ImageReference{
		CommunityGalleryImageID: imageRef.CommunityGalleryImageID,
		ID:                      id,
		Offer:                   offer,
		Publisher:               publisher,
		SKU:                     sku,
		SharedGalleryImageID:    imageRef.SharedGalleryImageID,
		Version:                 version,
	}
}

func CreateDiskResource(spec api.AzureProviderSpec, diskName string, vmID *string, plan *armcompute.Plan) *armcompute.Disk {
	var purchasePlan *armcompute.DiskPurchasePlan
	if plan != nil {
		purchasePlan = &armcompute.DiskPurchasePlan{
			Name:      plan.Name,
			Product:   plan.Product,
			Publisher: plan.Publisher,
		}
	}
	return &armcompute.Disk{
		Location: to.Ptr(spec.Location),
		Properties: &armcompute.DiskProperties{
			CreationData: &armcompute.CreationData{
				CreateOption: to.Ptr(armcompute.DiskCreateOptionEmpty),
			},
			DiskSizeGB:   pointer.Int32(spec.Properties.StorageProfile.OsDisk.DiskSizeGB),
			OSType:       to.Ptr(armcompute.OperatingSystemTypesLinux),
			PurchasePlan: purchasePlan,
			DiskState:    to.Ptr(armcompute.DiskStateAttached),
		},
		SKU: &armcompute.DiskSKU{
			Name: to.Ptr(armcompute.DiskStorageAccountTypes(spec.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType)),
		},
		Zones:     []*string{to.Ptr("1")},
		ManagedBy: vmID,
		Tags:      createResourceTags(spec.Tags),
		Name:      to.Ptr(diskName),
		Type:      to.Ptr("Microsoft.Compute/disks"),
	}
}

func createDataDisks(spec api.AzureProviderSpec, vmName string, deleteOption *armcompute.DiskDeleteOptionTypes) []*armcompute.DataDisk {
	specDataDisks := spec.Properties.StorageProfile.DataDisks
	if specDataDisks == nil {
		return nil
	}
	dataDisks := make([]*armcompute.DataDisk, 0, len(specDataDisks))
	for _, disk := range specDataDisks {
		d := &armcompute.DataDisk{
			CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesEmpty),
			Lun:          disk.Lun,
			Caching:      to.Ptr(armcompute.CachingTypes(disk.Caching)),
			DeleteOption: deleteOption,
			DiskSizeGB:   pointer.Int32(disk.DiskSizeGB),
			ManagedDisk: &armcompute.ManagedDiskParameters{
				StorageAccountType: to.Ptr(armcompute.StorageAccountTypes(disk.StorageAccountType)),
			},
			Name: to.Ptr(utils.CreateDataDiskName(vmName, disk)),
		}
		dataDisks = append(dataDisks, d)
	}
	return dataDisks
}

func createResourceTags(tags map[string]string) map[string]*string {
	vmTags := make(map[string]*string, len(tags))
	for k, v := range tags {
		vmTags[k] = to.Ptr(v)
	}
	return vmTags
}
