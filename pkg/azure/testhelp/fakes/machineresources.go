// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package fakes

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
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
	Name string
	// VM is the virtual machine.
	VM *armcompute.VirtualMachine
	// OSDisk is the OS disk associated to the VM.
	OSDisk *armcompute.Disk
	// DataDisks is the map of data disk name to Disk object.
	DataDisks map[string]*armcompute.Disk
	// NIC is the network interface associated to the VM.
	NIC *armnetwork.Interface
}

// CascadeDeleteOpts captures the cascade delete options for NIC, OSDisk and DataDisk.
type CascadeDeleteOpts struct {
	NIC      *armcompute.DeleteOptions
	OSDisk   *armcompute.DiskDeleteOptionTypes
	DataDisk *armcompute.DiskDeleteOptionTypes
}

// ShouldCascadeDeleteAllAttachedResources returns true if cascade delete is set for all NIC, OS/Data disks.
func (m *MachineResources) ShouldCascadeDeleteAllAttachedResources() bool {
	if m.VM != nil {
		nicDeleteOpt := GetCascadeDeleteOptForNIC(*m.VM)                 // A nil value indicates that there is no NIC
		osDiskDeleteOpt := GetCascadeDeleteOptForOsDisk(*m.VM)           // A nil value indicates that there is no OSDisk
		dataDisksDeleteOptsMap := GetCascadeDeleteOptForDataDisks(*m.VM) // An empty map means there are no DataDisks

		cascadeDeleteSetForNIC := nicDeleteOpt == nil || *nicDeleteOpt == armcompute.DeleteOptionsDelete
		cascadeDeleteSetForOSDisk := osDiskDeleteOpt == nil || *osDiskDeleteOpt == armcompute.DiskDeleteOptionTypesDelete
		cascadeDeleteSetForDataDisks := len(dataDisksDeleteOptsMap) == 0 || isCascadeDeleteSetForAllDataDisks(dataDisksDeleteOptsMap)

		return cascadeDeleteSetForNIC && cascadeDeleteSetForOSDisk && cascadeDeleteSetForDataDisks
	}
	return false
}

// HandleNICOnVMDelete detaches the NIC from the VM.
// This method will only be called if cascade delete for NIC is not turned on.
func (m *MachineResources) HandleNICOnVMDelete() {
	if m.VM != nil {
		nicDeleteOpt := GetCascadeDeleteOptForNIC(*m.VM)
		if nicDeleteOpt == nil || *nicDeleteOpt == armcompute.DeleteOptionsDelete {
			m.NIC = nil
		} else {
			m.NIC.Properties.VirtualMachine = nil
		}
	}
}

// HandleOSDiskOnVMDelete detaches the OSDisk from the VM.
// This method will only be called if cascade delete for OSDisk is not turned on.
func (m *MachineResources) HandleOSDiskOnVMDelete() {
	if m.VM != nil {
		osDiskDeleteOpt := GetCascadeDeleteOptForOsDisk(*m.VM)
		if osDiskDeleteOpt == nil || *osDiskDeleteOpt == armcompute.DiskDeleteOptionTypesDelete {
			m.OSDisk = nil
		} else {
			m.OSDisk.ManagedBy = nil
		}
	}
}

// HandleDataDisksOnVMDelete detaches the DataDisks from the VM.
// This method will only be called if cascade delete for DataDisk is not turned on.
func (m *MachineResources) HandleDataDisksOnVMDelete() {
	if m.VM != nil {
		diskDeleteOptMap := GetCascadeDeleteOptForDataDisks(*m.VM)
		for diskName, deleteOpt := range diskDeleteOptMap {
			if dataDisk, ok := m.DataDisks[diskName]; ok {
				if *deleteOpt == armcompute.DiskDeleteOptionTypesDelete {
					delete(m.DataDisks, diskName)
				} else {
					dataDisk.ManagedBy = nil
				}
			}
		}
		if len(m.DataDisks) == 0 {
			m.DataDisks = nil
		}
	}
}

// HasResources checks if the MachineResources object has any of VM, NIC, OSDisk, DataDisk resources.
// This will be used to just delete an instance of MachineResources when it has none of the resources.
func (m *MachineResources) HasResources() bool {
	return m.VM != nil || m.NIC != nil || m.OSDisk != nil || (m.DataDisks != nil && len(m.DataDisks) > 0)
}

// UpdateNICDeleteOpt updates the delete option for NIC.
func (m *MachineResources) UpdateNICDeleteOpt(deleteOpt *armcompute.DeleteOptions) {
	if m.VM != nil {
		if m.VM.Properties != nil && m.VM.Properties.NetworkProfile != nil && !utils.IsSliceNilOrEmpty(m.VM.Properties.NetworkProfile.NetworkInterfaces) {
			networkProperties := m.VM.Properties.NetworkProfile.NetworkInterfaces[0].Properties
			if networkProperties == nil {
				networkProperties = &armcompute.NetworkInterfaceReferenceProperties{}
			}
			networkProperties.DeleteOption = deleteOpt
		}
	}
}

// UpdateOSDiskDeleteOpt updates the delete option for OSDisk.
func (m *MachineResources) UpdateOSDiskDeleteOpt(deleteOpt *armcompute.DiskDeleteOptionTypes) {
	if m.VM != nil {
		if m.VM.Properties != nil && m.VM.Properties.StorageProfile != nil && m.VM.Properties.StorageProfile.OSDisk != nil {
			m.VM.Properties.StorageProfile.OSDisk.DeleteOption = deleteOpt
		}
	}
}

// UpdateDataDisksDeleteOpt updates the delete options for DataDisks.
func (m *MachineResources) UpdateDataDisksDeleteOpt(deleteOpt *armcompute.DiskDeleteOptionTypes) {
	if m.VM != nil {
		if m.VM.Properties != nil && m.VM.Properties.StorageProfile != nil && m.VM.Properties.StorageProfile.DataDisks != nil {
			for _, dataDisk := range m.VM.Properties.StorageProfile.DataDisks {
				dataDisk.DeleteOption = deleteOpt
			}
		}
	}
}

// AttachDataDisk attaches a data disk to the VM
func (m *MachineResources) AttachDataDisk(spec api.AzureProviderSpec, diskName string, deleteOption armcompute.DiskDeleteOptionTypes) error {
	if _, ok := m.DataDisks[diskName]; ok {
		return fmt.Errorf("disk %s already exists, cannot create a new disk with the same name", diskName)
	}
	dataDisk := createDataDisk(int32(len(m.DataDisks)+1), "None", &deleteOption, 20, testhelp.StorageAccountType, diskName)
	d := createDiskResource(spec, diskName, m.VM.ID, nil)
	m.DataDisks[diskName] = d
	m.VM.Properties.StorageProfile.DataDisks = append(m.VM.Properties.StorageProfile.DataDisks, dataDisk)
	return nil
}

func isCascadeDeleteSetForAllDataDisks(dataDiskDeleteOptsMap map[string]*armcompute.DiskDeleteOptionTypes) bool {
	for _, deleteOpt := range dataDiskDeleteOptsMap {
		if *deleteOpt != armcompute.DiskDeleteOptionTypesDelete {
			return false
		}
	}
	return true
}

// updateMachineResourcesFromVMParams updates MachineResources from already built vmParams and ProviderSpec.
// This function would typically be used to create MachineResources in the CreateMachine driver call flow where
// it is assumed that NIC creation will be done first which will already create a MachineResource. This function will
// then create the rest of the resources and also update the NIC to refer to the VM ID.
func updateMachineResourcesFromVMParams(spec api.AzureProviderSpec, resourceGroup string, vmParams armcompute.VirtualMachine, machineResources *MachineResources) {
	vmName := *vmParams.Name
	newVM := vmParams
	newVM.ID = to.Ptr(CreateVirtualMachineID(testhelp.SubscriptionID, resourceGroup, vmName))
	machineResources.VM = &newVM
	if machineResources.NIC != nil {
		if machineResources.NIC.Properties.VirtualMachine == nil {
			machineResources.NIC.Properties.VirtualMachine = &armnetwork.SubResource{}
		}
		machineResources.NIC.Properties.VirtualMachine.ID = newVM.ID
	}
	osDisk := createDiskResource(spec, utils.CreateOSDiskName(vmName), newVM.ID, newVM.Plan)
	dataDisks := createDataDiskResources(spec, newVM.ID, vmName)
	machineResources.OSDisk = osDisk
	machineResources.DataDisks = dataDisks
}

//----------------------------------------------------------------------
// Builder for MachineResources
// This builder should not be used if CreateMachine driver method is
// being tested. The CreateMachine already populates armcompute.VirtualMachine.
// If one wishes to create MachineResources from armcompute.VirtualMachine then
// use function updateMachineResourcesFromVMParams instead.
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

// WithPlan initializes MachineResources with a plan.
func (b *MachineResourcesBuilder) WithPlan(plan armcompute.Plan) *MachineResourcesBuilder {
	b.plan = &plan
	return b
}

// WithCascadeDeleteOptions initializes MachineResources with cascade delete options for NIC, OS/Data disks.
func (b *MachineResourcesBuilder) WithCascadeDeleteOptions(opts CascadeDeleteOpts) *MachineResourcesBuilder {
	b.cascadeDeleteOpts = &opts
	if b.cascadeDeleteOpts.NIC == nil {
		b.cascadeDeleteOpts.NIC = to.Ptr(armcompute.DeleteOptionsDetach)
	}
	if b.cascadeDeleteOpts.OSDisk == nil {
		b.cascadeDeleteOpts.OSDisk = to.Ptr(armcompute.DiskDeleteOptionTypesDetach)
	}
	if b.cascadeDeleteOpts.DataDisk == nil {
		b.cascadeDeleteOpts.DataDisk = to.Ptr(armcompute.DiskDeleteOptionTypesDetach)
	}
	return b
}

// BuildAllResources creates a MachineResources object creating VM, NIC, OSDisk and DataDisks.
func (b *MachineResourcesBuilder) BuildAllResources() MachineResources {
	return b.BuildWith(true, true, true, true, nil)
}

// BuildWith creates a MachineResources object optionally creating resources as indicated by the method arguments.
func (b *MachineResourcesBuilder) BuildWith(createVM, createNIC, createOSDisk, createDataDisk bool, withNonExistentVMID *string) MachineResources {
	if b.cascadeDeleteOpts == nil {
		b.cascadeDeleteOpts = &CascadeDeleteAllResources
	}
	return b.createMachineResources(createVM, createNIC, createOSDisk, createDataDisk, withNonExistentVMID)
}

// createMachineResources creates MachineResources object optionally creating resources as indicated by the method arguments.
func (b *MachineResourcesBuilder) createMachineResources(createVM, createNIC, createOSDisk, createDataDisks bool, nonExistentVMID *string) MachineResources {
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
		nic = createNICResource(b.spec, vmID, utils.CreateNICName(b.vmName))
	}
	if createOSDisk {
		osDisk = createDiskResource(b.spec, utils.CreateOSDiskName(b.vmName), vmID, b.plan)
	}
	if createDataDisks {
		dataDisks = createDataDiskResources(b.spec, vmID, b.vmName)
	}
	return MachineResources{
		Name:      b.vmName,
		VM:        vm,
		OSDisk:    osDisk,
		DataDisks: dataDisks,
		NIC:       nic,
	}
}

func createDataDiskResources(spec api.AzureProviderSpec, vmID *string, vmName string) map[string]*armcompute.Disk {
	specDataDisks := spec.Properties.StorageProfile.DataDisks
	dataDisks := make(map[string]*armcompute.Disk, len(specDataDisks))
	for _, specDataDisk := range specDataDisks {
		diskName := utils.CreateDataDiskName(vmName, specDataDisk.Name, specDataDisk.Lun)
		dataDisks[diskName] = createDiskResource(spec, diskName, vmID, nil)
	}
	return dataDisks
}

func createNICResource(spec api.AzureProviderSpec, vmID *string, nicName string) *armnetwork.Interface {
	ipConfigID := CreateIPConfigurationID(testhelp.SubscriptionID, spec.ResourceGroup, nicName, nicName)
	interfaceID := CreateNetworkInterfaceID(testhelp.SubscriptionID, spec.ResourceGroup, nicName)

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
		Tags: utils.CreateResourceTags(spec.Tags),
		ID:   &interfaceID,
		Name: &nicName,
		Type: to.Ptr("Microsoft.Network/networkInterfaces"),
	}
}

func createVMResource(spec api.AzureProviderSpec, vmName string, plan *armcompute.Plan, cascadeDeleteOpts *CascadeDeleteOpts) *armcompute.VirtualMachine {
	id := CreateVirtualMachineID(testhelp.SubscriptionID, spec.ResourceGroup, vmName)
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
		Tags:  utils.CreateResourceTags(spec.Tags),
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

func createDiskResource(spec api.AzureProviderSpec, diskName string, vmID *string, plan *armcompute.Plan) *armcompute.Disk {
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
		Tags:      utils.CreateResourceTags(spec.Tags),
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
		diskName := utils.CreateDataDiskName(vmName, disk.Name, disk.Lun)
		d := createDataDisk(disk.Lun, armcompute.CachingTypes(disk.Caching), deleteOption, disk.DiskSizeGB, armcompute.StorageAccountTypes(disk.StorageAccountType), diskName)
		dataDisks = append(dataDisks, d)
	}
	return dataDisks
}

func createDataDisk(lun int32, caching armcompute.CachingTypes, deleteOption *armcompute.DiskDeleteOptionTypes, diskSize int32, storageAccountType armcompute.StorageAccountTypes, diskName string) *armcompute.DataDisk {
	return &armcompute.DataDisk{
		CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesEmpty),
		Lun:          to.Ptr(lun),
		Caching:      to.Ptr(caching),
		DeleteOption: deleteOption,
		DiskSizeGB:   pointer.Int32(diskSize),
		ManagedDisk: &armcompute.ManagedDiskParameters{
			StorageAccountType: to.Ptr(storageAccountType),
		},
		Name: to.Ptr(diskName),
	}
}
