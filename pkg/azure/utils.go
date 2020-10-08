package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest/to"
	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	clientutils "github.com/gardener/machine-controller-manager-provider-azure/pkg/client"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/apimachinery/pkg/runtime"
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

func dependencyNameFromVMName(vmName, suffix string) string {
	return vmName + suffix
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

func (d *Driver) getNICParameters(vmName string, subnet *network.Subnet) network.Interface {

	var (
		nicName            = dependencyNameFromVMName(vmName, nicSuffix)
		location           = d.AzureProviderSpec.Location
		enableIPForwarding = true
	)

	// Add tags to the machine resources
	tagList := map[string]*string{}
	for idx, element := range d.AzureProviderSpec.Tags {
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
						Subnet:                    subnet,
					},
				},
			},
			EnableIPForwarding:          &enableIPForwarding,
			EnableAcceleratedNetworking: d.AzureProviderSpec.Properties.NetworkProfile.AcceleratedNetworking,
		},
		Tags: tagList,
	}

	return NICParameters
}

func (d *Driver) generateDataDisks(vmName string, azureDataDisks []api.AzureDataDisk) []compute.DataDisk {
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

func (d *Driver) getVMParameters(vmName string, image *compute.VirtualMachineImage, networkInterfaceReferenceID string) compute.VirtualMachine {

	var (
		diskName    = dependencyNameFromVMName(vmName, diskSuffix)
		UserDataEnc = base64.StdEncoding.EncodeToString([]byte(d.Secret.Data["userData"]))
		location    = d.AzureProviderSpec.Location
	)

	// Add tags to the machine resources
	tagList := map[string]*string{}
	for idx, element := range d.AzureProviderSpec.Tags {
		tagList[idx] = to.StringPtr(element)
	}

	imageReference := getImageReference(d)

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
				VMSize: compute.VirtualMachineSizeTypes(d.AzureProviderSpec.Properties.HardwareProfile.VMSize),
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &imageReference,
				OsDisk: &compute.OSDisk{
					Name:    &diskName,
					Caching: compute.CachingTypes(d.AzureProviderSpec.Properties.StorageProfile.OsDisk.Caching),
					ManagedDisk: &compute.ManagedDiskParameters{
						StorageAccountType: compute.StorageAccountTypes(d.AzureProviderSpec.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType),
					},
					DiskSizeGB:   &d.AzureProviderSpec.Properties.StorageProfile.OsDisk.DiskSizeGB,
					CreateOption: compute.DiskCreateOptionTypes(d.AzureProviderSpec.Properties.StorageProfile.OsDisk.CreateOption),
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  &vmName,
				AdminUsername: &d.AzureProviderSpec.Properties.OsProfile.AdminUsername,
				CustomData:    &UserDataEnc,
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: &d.AzureProviderSpec.Properties.OsProfile.LinuxConfiguration.DisablePasswordAuthentication,
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								Path:    &d.AzureProviderSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.Path,
								KeyData: &d.AzureProviderSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.KeyData,
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

	if d.AzureProviderSpec.Properties.StorageProfile.DataDisks != nil && len(d.AzureProviderSpec.Properties.StorageProfile.DataDisks) > 0 {
		dataDisks := d.generateDataDisks(vmName, d.AzureProviderSpec.Properties.StorageProfile.DataDisks)
		VMParameters.StorageProfile.DataDisks = &dataDisks
	}

	if d.AzureProviderSpec.Properties.Zone != nil {
		VMParameters.Zones = &[]string{strconv.Itoa(*d.AzureProviderSpec.Properties.Zone)}
	} else if d.AzureProviderSpec.Properties.AvailabilitySet != nil {
		VMParameters.VirtualMachineProperties.AvailabilitySet = &compute.SubResource{
			ID: &d.AzureProviderSpec.Properties.AvailabilitySet.ID,
		}
	}

	if d.AzureProviderSpec.Properties.IdentityID != nil && *d.AzureProviderSpec.Properties.IdentityID != "" {
		VMParameters.Identity = &compute.VirtualMachineIdentity{
			Type: compute.ResourceIdentityTypeUserAssigned,
			UserAssignedIdentities: map[string]*compute.VirtualMachineIdentityUserAssignedIdentitiesValue{
				*d.AzureProviderSpec.Properties.IdentityID: {},
			},
		}
	}

	return VMParameters
}

func getImageReference(d *Driver) compute.ImageReference {
	imageRefClass := d.AzureProviderSpec.Properties.StorageProfile.ImageReference
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

func (d *Driver) createVMNicDisk(req *driver.CreateMachineRequest) (*compute.VirtualMachine, error) {

	providerSpec, err := decodeProviderSpecAndSecret(req.MachineClass, req.Secret)
	d.AzureProviderSpec = providerSpec
	if err != nil {
		return nil, err
	}

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

	clients, err := d.SPI.Setup(req.Secret)
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

	/*
		Subnet fetching
	*/
	// Getting the subnet object for subnetName
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

	/*
		NIC creation
	*/

	// Creating NICParameters for new NIC creation request
	NICParameters := d.getNICParameters(vmName, &subnet)

	// NIC creation request
	NICFuture, err := clients.Nic.CreateOrUpdate(ctx, resourceGroupName, *NICParameters.Name, NICParameters)
	if err != nil {
		// Since machine creation failed, delete any infra resources created
		deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, clientutils.OnARMAPIErrorFail(prometheusServiceNIC, err, "NIC.CreateOrUpdate failed for %s", *NICParameters.Name)
	}

	// Wait until NIC is created
	err = NICFuture.WaitForCompletionRef(ctx, clients.Nic.Client)
	if err != nil {
		// Since machine creation failed, delete any infra resources created
		deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, clientutils.OnARMAPIErrorFail(prometheusServiceNIC, err, "NIC.WaitForCompletionRef failed for %s", *NICParameters.Name)
	}
	clientutils.OnARMAPISuccess(prometheusServiceNIC, "NIC.CreateOrUpdate")

	// Fetch NIC details
	NIC, err := NICFuture.Result(clients.Nic)
	if err != nil {
		// Since machine creation failed, delete any infra resources created
		deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, err
	}

	/*
		VM creation
	*/
	startTime := time.Now()
	imageRefClass := d.AzureProviderSpec.Properties.StorageProfile.ImageReference
	// if ID is not set the image is referenced using a URN
	if imageRefClass.ID == "" {

		imageReference := getImageReference(d)
		vmImage, err := clients.Images.Get(
			ctx,
			d.AzureProviderSpec.Location,
			*imageReference.Publisher,
			*imageReference.Offer,
			*imageReference.Sku,
			*imageReference.Version)

		if err != nil {
			//Since machine creation failed, delete any infra resources created
			deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
			if deleteErr != nil {
				klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
			}

			return nil, clientutils.OnARMAPIErrorFail(prometheusServiceVM, err, "VirtualMachineImagesclientutils.Get failed for %s", req.MachineClass.Name)
		}

		if vmImage.Plan != nil {
			// If VMImage.Plan exists, check if agreement is accepted and if not accept it for the subscription

			agreement, err := clients.Marketplace.Get(
				ctx,
				*vmImage.Plan.Publisher,
				*vmImage.Plan.Product,
				*vmImage.Plan.Name,
			)

			if err != nil {
				//Since machine creation failed, delete any infra resources created
				deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
				if deleteErr != nil {
					klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
				}

				return nil, clientutils.OnARMAPIErrorFail(prometheusServiceVM, err, "MarketplaceAgreementsclient.Get failed for %s", req.MachineClass.Name)
			}

			if agreement.Accepted == nil || *agreement.Accepted == false {
				// Need to accept the terms at least once for the subscription
				klog.V(2).Info("Accepting terms for subscription to make use of the plan")

				agreement.Accepted = to.BoolPtr(true)
				_, err = clients.Marketplace.Create(
					ctx,
					*vmImage.Plan.Publisher,
					*vmImage.Plan.Product,
					*vmImage.Plan.Name,
					agreement,
				)

				if err != nil {
					//Since machine creation failed, delete any infra resources created
					deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
					if deleteErr != nil {
						klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
					}

					return nil, clientutils.OnARMAPIErrorFail(prometheusServiceVM, err, "MarketplaceAgreementsclientutils.Create failed for %s", req.MachineClass.Name)
				}
			}
		}

		vmImageRef = &vmImage
	}

	// Creating VMParameters for new VM creation request
	VMParameters := d.getVMParameters(vmName, vmImageRef, *NIC.ID)

	// VM creation request
	VMFuture, err := clients.VM.CreateOrUpdate(ctx, resourceGroupName, *VMParameters.Name, VMParameters)
	if err != nil {
		//Since machine creation failed, delete any infra resources created
		deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, clientutils.OnARMAPIErrorFail(prometheusServiceVM, err, "VM.CreateOrUpdate failed for %s", *VMParameters.Name)
	}

	// Wait until VM is created
	err = VMFuture.WaitForCompletionRef(ctx, clients.VM.Client)
	if err != nil {
		// Since machine creation failed, delete any infra resources created
		deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, clientutils.OnARMAPIErrorFail(prometheusServiceVM, err, "VM.WaitForCompletionRef failed for %s", *VMParameters.Name)
	}
	klog.Infof("VM Created in %d", time.Now().Sub(startTime))

	// Fetch VM details
	VM, err := VMFuture.Result(clients.VM)
	if err != nil {
		// Since machine creation failed, delete any infra resources created
		deleteErr := clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, clientutils.OnARMAPIErrorFail(prometheusServiceVM, err, "VMFuture.Result failed for %s", *VMParameters.Name)
	}
	clientutils.OnARMAPISuccess(prometheusServiceVM, "VM.CreateOrUpdate")

	return &VM, nil
}

func fillUpMachineClass(azureMachineClass *v1alpha1.AzureMachineClass, machineClass *v1alpha1.MachineClass) error {
	var (
		err        error
		properties api.AzureVirtualMachineProperties
		subnetInfo api.AzureSubnetInfo
	)

	// Extract the Properties object from the AzureMachineClass
	// to fill it up in the MachineClass
	data, _ := json.Marshal(azureMachineClass.Spec.Properties)
	err = json.Unmarshal(data, &properties)

	// Extract the Subnet Info object form the AzureMachineClass
	// to fill it up in the MachineClass
	data, _ = json.Marshal(azureMachineClass.Spec.SubnetInfo)
	err = json.Unmarshal(data, &subnetInfo)

	providerSpec := &api.AzureProviderSpec{
		Location:      azureMachineClass.Spec.Location,
		Tags:          azureMachineClass.Spec.Tags,
		Properties:    properties,
		ResourceGroup: azureMachineClass.Spec.ResourceGroup,
		SubnetInfo:    subnetInfo,
	}

	// Marshal providerSpec into Raw Bytes
	providerSpecMarshal, err := json.Marshal(providerSpec)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	machineClass.SecretRef = azureMachineClass.Spec.SecretRef
	machineClass.Name = azureMachineClass.Name
	machineClass.Labels = azureMachineClass.Labels
	machineClass.Annotations = azureMachineClass.Annotations
	machineClass.Finalizers = azureMachineClass.Finalizers
	machineClass.ProviderSpec = runtime.RawExtension{
		Raw: providerSpecMarshal,
	}

	return err
}
