package mock

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest/to"
	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	clientutils "github.com/gardener/machine-controller-manager-provider-azure/pkg/client"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog"
)

const (
	nicSuffix      = "-nic"
	diskSuffix     = "-os-disk"
	dataDiskSuffix = "-data-disk"
)

const (
	prometheusServiceSubnet = "subnet"
	prometheusServiceVM     = "virtual_machine"
	prometheusServiceNIC    = "network_interfaces"
	prometheusServiceDisk   = "disks"
)

func encodeMachineID(location, vmName string) string {
	return fmt.Sprintf("azure:///%s/%s", location, vmName)
}

func dependencyNameFromVMNameAndDependency(dependency, vmName, suffix string) string {
	return vmName + "-" + dependency + suffix
}

func getAzureDataDiskPrefix(name string, lun *int32) string {
	if name != "" {
		return fmt.Sprintf("%s-%d", name, *lun)
	}
	return fmt.Sprintf("%d", *lun)
}

func getAzureDataDiskNames(azureDataDisks []api.AzureDataDisk, vmname, suffix string) []string {
	azureDataDiskNames := make([]string, len(azureDataDisks))
	for i, disk := range azureDataDisks {
		var diskLun *int32
		if disk.Lun != nil {
			diskLun = disk.Lun
		} else {
			lun := int32(i)
			diskLun = &lun
		}
		azureDataDiskNames[i] = dependencyNameFromVMNameAndDependency(getAzureDataDiskPrefix(disk.Name, diskLun), vmname, suffix)
	}
	return azureDataDiskNames
}

// Get ...
func (client SubnetsClient) Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, expand string) (*network.Subnet, error) {
	return &network.Subnet{Name: &subnetName}, nil
}

// CreateOrUpdate ...
func (client InterfacesClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, networkInterfaceName string, parameters network.Interface) (InterfacesClient, error) {
	var result InterfacesClient
	result.ID = networkInterfaceName
	return result, nil
}

func dependencyNameFromVMName(vmName, suffix string) string {
	return vmName + suffix
}

func (ms *PluginSPIImpl) getNICParameters(vmName string, subnet network.Subnet) network.Interface {

	var (
		nicName            = dependencyNameFromVMName(vmName, nicSuffix)
		location           = ms.AzureProviderSpec.Location
		enableIPForwarding = true
	)

	// Add tags to the machine resources
	tagList := map[string]*string{}
	for idx, element := range ms.AzureProviderSpec.Tags {
		tagList[idx] = to.StringPtr(element)
	}

	NICParameters := network.Interface{
		Name:     &nicName,
		Location: &location,
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: &nicName,
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: network.Dynamic,
						Subnet:                    &subnet,
					},
				},
			},
			EnableIPForwarding:          &enableIPForwarding,
			EnableAcceleratedNetworking: ms.AzureProviderSpec.Properties.NetworkProfile.AcceleratedNetworking,
		},
		Tags: tagList,
	}

	return NICParameters
}

func (ms *PluginSPIImpl) createVMNicDisk(req *driver.CreateMachineRequest) (*compute.VirtualMachine, error) {
	providerSpec, err := decodeProviderSpecAndSecret(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}
	ms.AzureProviderSpec = providerSpec

	var (
		ctx               = context.Background()
		vmName            = strings.ToLower(req.Machine.Name)
		resourceGroupName = providerSpec.ResourceGroup
		vnetName          = providerSpec.SubnetInfo.VnetName
		vnetResourceGroup = resourceGroupName
		subnetName        = providerSpec.SubnetInfo.SubnetName
		nicName           = dependencyNameFromVMName(vmName, nicSuffix)
		diskName          = dependencyNameFromVMName(vmName, diskSuffix)
		vmImageRef        *compute.VirtualMachineImage
	)

	clients, err := ms.SPI.Setup(req.Secret)
	if err != nil {
		return nil, err
	}

	// Check if the machine should assigned to a vnet in a different resource group.
	if providerSpec.SubnetInfo.VnetResourceGroup != nil {
		vnetResourceGroup = *providerSpec.SubnetInfo.VnetResourceGroup
	}

	var dataDiskNames []string
	if providerSpec.Properties.StorageProfile.DataDisks != nil && len(providerSpec.Properties.StorageProfile.DataDisks) > 0 {
		dataDiskNames = getAzureDataDiskNames(providerSpec.Properties.StorageProfile.DataDisks, vmName, dataDiskSuffix)
	}

	subnet, err := clients.Subnet.Get(
		ctx,
		vnetResourceGroup,
		vnetName,
		subnetName,
		"",
	)
	if err != nil {
		return nil, clientutils.OnARMAPIErrorFail(prometheusServiceSubnet, err, "Subnet.Get failed for %s due to %s", subnetName, err)
	}
	clientutils.OnARMAPISuccess(prometheusServiceSubnet, "subnet.Get")

	NICParameters := ms.getNICParameters(vmName, *subnet)

	// NIC creation request
	NIC, err := clients.Nic.CreateOrUpdate(ctx, resourceGroupName, *NICParameters.Name, NICParameters)
	if err != nil {
		// Since machine creation failed, delete any infra resources created
		deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, clientutils.OnARMAPIErrorFail(prometheusServiceNIC, err, "NIC.CreateOrUpdate failed for %s", *NICParameters.Name)
	}

	/*
		VM creation
	*/

	// Creating VMParameters for new VM creation request
	VMParameters := ms.getVMParameters(vmName, vmImageRef, NIC.ID)

	// VM creation request
	VM, err := clients.VM.CreateOrUpdate(ctx, resourceGroupName, *VMParameters.Name, VMParameters)
	if err != nil {
		//Since machine creation failed, delete any infra resources created
		deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, clientutils.OnARMAPIErrorFail(prometheusServiceVM, err, "VM.CreateOrUpdate failed for %s", *VMParameters.Name)
	}
	return VM, nil

}

// CreateMachine ...
func (ms *PluginSPIImpl) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	ms.Secret = req.Secret
	virtualMachine, err := ms.createVMNicDisk(req)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	providerID := encodeMachineID(*virtualMachine.Location, *virtualMachine.Name)
	return &driver.CreateMachineResponse{ProviderID: providerID, NodeName: req.Machine.Name}, nil
}

func getImageReference(ms *PluginSPIImpl) compute.ImageReference {
	imageRefClass := ms.AzureProviderSpec.Properties.StorageProfile.ImageReference
	if imageRefClass.ID != "" {
		return compute.ImageReference{
			ID: &imageRefClass.ID,
		}
	}

	splits := strings.Split(*imageRefClass.URN, ":")
	publisher := splits[0]
	offer := splits[1]
	sku := splits[2]
	version := splits[3]
	return compute.ImageReference{
		Publisher: &publisher,
		Offer:     &offer,
		Sku:       &sku,
		Version:   &version,
	}
}

func (ms *PluginSPIImpl) getVMParameters(vmName string, image *compute.VirtualMachineImage, networkInterfaceReferenceID string) compute.VirtualMachine {

	var (
		diskName    = dependencyNameFromVMName(vmName, diskSuffix)
		UserDataEnc = base64.StdEncoding.EncodeToString([]byte(ms.Secret.Data["userData"]))
		location    = ms.AzureProviderSpec.Location
	)

	// Add tags to the machine resources
	tagList := map[string]*string{}
	for idx, element := range ms.AzureProviderSpec.Tags {
		tagList[idx] = to.StringPtr(element)
	}

	imageReference := getImageReference(ms)

	var plan *compute.Plan
	if image != nil && image.Plan != nil {
		// If image.Plan exists, create a plan object and attach it to the VM
		klog.V(2).Infof("Creating a plan object and attaching it to the VM - %q", vmName)
		plan = &compute.Plan{
			Name:      image.VirtualMachineImageProperties.Plan.Name,
			Product:   image.VirtualMachineImageProperties.Plan.Product,
			Publisher: image.VirtualMachineImageProperties.Plan.Publisher,
		}
	}

	VMParameters := compute.VirtualMachine{
		Name:     &vmName,
		Plan:     plan,
		Location: &location,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(ms.AzureProviderSpec.Properties.HardwareProfile.VMSize),
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &imageReference,
				OsDisk: &compute.OSDisk{
					Name:    &diskName,
					Caching: compute.CachingTypes(ms.AzureProviderSpec.Properties.StorageProfile.OsDisk.Caching),
					ManagedDisk: &compute.ManagedDiskParameters{
						StorageAccountType: compute.StorageAccountTypes(ms.AzureProviderSpec.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType),
					},
					DiskSizeGB:   &ms.AzureProviderSpec.Properties.StorageProfile.OsDisk.DiskSizeGB,
					CreateOption: compute.DiskCreateOptionTypes(ms.AzureProviderSpec.Properties.StorageProfile.OsDisk.CreateOption),
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  &vmName,
				AdminUsername: &ms.AzureProviderSpec.Properties.OsProfile.AdminUsername,
				CustomData:    &UserDataEnc,
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: &ms.AzureProviderSpec.Properties.OsProfile.LinuxConfiguration.DisablePasswordAuthentication,
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								Path:    &ms.AzureProviderSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.Path,
								KeyData: &ms.AzureProviderSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.KeyData,
							},
						},
					},
				},
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: &networkInterfaceReferenceID,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
		},
		Tags: tagList,
	}

	if ms.AzureProviderSpec.Properties.StorageProfile.DataDisks != nil && len(ms.AzureProviderSpec.Properties.StorageProfile.DataDisks) > 0 {
		dataDisks := ms.generateDataDisks(vmName, ms.AzureProviderSpec.Properties.StorageProfile.DataDisks)
		VMParameters.StorageProfile.DataDisks = &dataDisks
	}

	if ms.AzureProviderSpec.Properties.Zone != nil {
		VMParameters.Zones = &[]string{strconv.Itoa(*ms.AzureProviderSpec.Properties.Zone)}
	} else if ms.AzureProviderSpec.Properties.AvailabilitySet != nil {
		VMParameters.VirtualMachineProperties.AvailabilitySet = &compute.SubResource{
			ID: &ms.AzureProviderSpec.Properties.AvailabilitySet.ID,
		}
	}

	if ms.AzureProviderSpec.Properties.IdentityID != nil && *ms.AzureProviderSpec.Properties.IdentityID != "" {
		VMParameters.Identity = &compute.VirtualMachineIdentity{
			Type: compute.ResourceIdentityTypeUserAssigned,
			UserAssignedIdentities: map[string]*compute.VirtualMachineIdentityUserAssignedIdentitiesValue{
				*ms.AzureProviderSpec.Properties.IdentityID: {},
			},
		}
	}

	return VMParameters
}

func (ms *PluginSPIImpl) generateDataDisks(vmName string, azureDataDisks []api.AzureDataDisk) []compute.DataDisk {
	var dataDisks []compute.DataDisk
	for i, azureDataDisk := range azureDataDisks {

		var dataDiskLun *int32
		if azureDataDisk.Lun != nil {
			dataDiskLun = azureDataDisk.Lun
		} else {
			lun := int32(i)
			dataDiskLun = &lun
		}

		dataDiskName := dependencyNameFromVMNameAndDependency(getAzureDataDiskPrefix(azureDataDisk.Name, dataDiskLun), vmName, dataDiskSuffix)

		var caching compute.CachingTypes
		if azureDataDisk.Caching != "" {
			caching = compute.CachingTypes(azureDataDisk.Caching)
		} else {
			caching = compute.CachingTypesNone
		}

		dataDiskSize := azureDataDisk.DiskSizeGB

		dataDisk := compute.DataDisk{
			Lun:     dataDiskLun,
			Name:    &dataDiskName,
			Caching: caching,
			ManagedDisk: &compute.ManagedDiskParameters{
				StorageAccountType: compute.StorageAccountTypes(azureDataDisk.StorageAccountType),
			},
			DiskSizeGB:   &dataDiskSize,
			CreateOption: compute.DiskCreateOptionTypesEmpty,
		}
		dataDisks = append(dataDisks, dataDisk)
	}
	return dataDisks
}

// DeleteVMNicDisks deletes the VM and associated Disks and NIC
func (clients *FakeAzureDriverClients) DeleteVMNicDisks(ctx context.Context, resourceGroupName string, VMName string, nicName string, diskName string, dataDiskNames []string) error {
	return nil
}
