/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package azure contains the cloud provider specific implementations to manage machines
package azure

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	spi "github.com/gardener/machine-controller-manager-provider-azure/pkg/spi"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	backoff "github.com/gardener/machine-controller-manager/pkg/util/backoff"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	metrics "github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

// constant suffixes
const (
	nicSuffix      = "-nic"
	diskSuffix     = "-os-disk"
	dataDiskSuffix = "-data-disk"
)

// constant services
const (
	prometheusServiceSubnet = "subnet"
	prometheusServiceVM     = "virtual_machine"
	prometheusServiceNIC    = "network_interfaces"
	prometheusServiceDisk   = "disks"
)

// constant intervals for backoff/retry mechanism
const (
	intialInterval     = 10 * time.Second
	maxInterval        = 2 * time.Minute
	maxElapsedTime     = 10 * time.Minute
	nicDeletionTimeout = 10 * time.Minute
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
		var diskLun *int32 = disk.Lun
		azureDataDiskNames[i] = dependencyNameFromVMNameAndDependency(getAzureDataDiskPrefix(disk.Name, diskLun), vmname, suffix)
	}
	return azureDataDiskNames
}

func getNICParameters(vmName string, subnet *network.Subnet, providerSpec *api.AzureProviderSpec) network.Interface {

	var (
		nicName            = dependencyNameFromVMName(vmName, nicSuffix)
		location           = providerSpec.Location
		enableIPForwarding = true
	)

	// Add tags to the machine resources
	tagList := map[string]*string{}
	for idx, element := range providerSpec.Tags {
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
						PrivateIPAllocationMethod: network.IPAllocationMethodDynamic,
						Subnet:                    subnet,
					},
				},
			},
			EnableIPForwarding:          &enableIPForwarding,
			EnableAcceleratedNetworking: providerSpec.Properties.NetworkProfile.AcceleratedNetworking,
		},
		Tags: tagList,
	}

	return NICParameters
}

func generateDataDisks(vmName string, azureDataDisks []api.AzureDataDisk) []compute.DataDisk {
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

func generateSSHAuthorizedKeys(privateKey *rsa.PrivateKey) ([]byte, error) {
	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	publicKey := ssh.MarshalAuthorizedKey(pubKey)
	return bytes.Trim(publicKey, "\x0a"), nil
}

func generateDummyPublicKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", err
	}

	sshPublicKey, err := generateSSHAuthorizedKeys(privateKey)
	if err != nil {
		return "", err
	}

	return string(sshPublicKey), nil
}

func getVMParameters(vmName string, image *compute.VirtualMachineImage, networkInterfaceReferenceID string, providerSpec *api.AzureProviderSpec, secret *corev1.Secret) (compute.VirtualMachine, error) {

	var (
		diskName    = dependencyNameFromVMName(vmName, diskSuffix)
		UserDataEnc = base64.StdEncoding.EncodeToString([]byte(secret.Data["userData"]))
		location    = providerSpec.Location
	)

	// Add tags to the machine resources
	tagList := map[string]*string{}
	for idx, element := range providerSpec.Tags {
		tagList[idx] = to.StringPtr(element)
	}

	imageReference := getImageReference(providerSpec)

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
				VMSize: compute.VirtualMachineSizeTypes(providerSpec.Properties.HardwareProfile.VMSize),
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &imageReference,
				OsDisk: &compute.OSDisk{
					Name:    &diskName,
					Caching: compute.CachingTypes(providerSpec.Properties.StorageProfile.OsDisk.Caching),
					ManagedDisk: &compute.ManagedDiskParameters{
						StorageAccountType: compute.StorageAccountTypes(providerSpec.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType),
					},
					DiskSizeGB:   &providerSpec.Properties.StorageProfile.OsDisk.DiskSizeGB,
					CreateOption: compute.DiskCreateOptionTypes(providerSpec.Properties.StorageProfile.OsDisk.CreateOption),
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  &vmName,
				AdminUsername: &providerSpec.Properties.OsProfile.AdminUsername,
				CustomData:    &UserDataEnc,
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: &providerSpec.Properties.OsProfile.LinuxConfiguration.DisablePasswordAuthentication,
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								Path:    &providerSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.Path,
								KeyData: &providerSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.KeyData,
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

	if providerSpec.Properties.StorageProfile.DataDisks != nil && len(providerSpec.Properties.StorageProfile.DataDisks) > 0 {
		dataDisks := generateDataDisks(vmName, providerSpec.Properties.StorageProfile.DataDisks)
		VMParameters.StorageProfile.DataDisks = &dataDisks
	}

	if providerSpec.Properties.Zone != nil {
		VMParameters.Zones = &[]string{strconv.Itoa(*providerSpec.Properties.Zone)}
	}

	// DEPRECATED: This will be removed in future in favour of the machineSet field which has a type for AvailabilitySet.
	// TODO: Remove in future release.
	if providerSpec.Properties.AvailabilitySet != nil {
		VMParameters.VirtualMachineProperties.AvailabilitySet = &compute.SubResource{
			ID: &providerSpec.Properties.AvailabilitySet.ID,
		}
	}

	if providerSpec.Properties.MachineSet != nil {
		switch providerSpec.Properties.MachineSet.Kind {
		case machine.MachineSetKindVMO:
			VMParameters.VirtualMachineProperties.VirtualMachineScaleSet = &compute.SubResource{
				ID: &providerSpec.Properties.MachineSet.ID,
			}
		case machine.MachineSetKindAvailabilitySet:
			VMParameters.VirtualMachineProperties.AvailabilitySet = &compute.SubResource{
				ID: &providerSpec.Properties.MachineSet.ID,
			}
		}
	}

	if providerSpec.Properties.IdentityID != nil && *providerSpec.Properties.IdentityID != "" {
		VMParameters.Identity = &compute.VirtualMachineIdentity{
			Type: compute.ResourceIdentityTypeUserAssigned,
			UserAssignedIdentities: map[string]*compute.VirtualMachineIdentityUserAssignedIdentitiesValue{
				*providerSpec.Properties.IdentityID: {},
			},
		}
	}

	if len(providerSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.KeyData) == 0 {
		// We create a dummy SSH Public key, since it is required for VM creation to have a public key
		publicKey, err := generateDummyPublicKey()
		if err != nil {
			return compute.VirtualMachine{}, err
		}

		VMParameters.VirtualMachineProperties.OsProfile.LinuxConfiguration.SSH.PublicKeys = &[]compute.SSHPublicKey{
			{
				Path:    &providerSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.Path,
				KeyData: &publicKey,
			},
		}
	}

	return VMParameters, nil
}

func getImageReference(providerSpec *api.AzureProviderSpec) compute.ImageReference {
	imageRefClass := providerSpec.Properties.StorageProfile.ImageReference
	if imageRefClass.ID != "" {
		return compute.ImageReference{
			ID: &imageRefClass.ID,
		}
	}

	if imageRefClass.CommunityGalleryImageID != nil {
		return compute.ImageReference{
			CommunityGalleryImageID: imageRefClass.CommunityGalleryImageID,
		}
	}

	if imageRefClass.SharedGalleryImageID != nil {
		return compute.ImageReference{
			SharedGalleryImageID: imageRefClass.SharedGalleryImageID,
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

func (d *MachinePlugin) createVMNicDisk(req *driver.CreateMachineRequest) (*compute.VirtualMachine, error) {

	providerSpec, err := DecodeProviderSpecAndSecret(req.MachineClass, req.Secret)
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

	// get the azuredriverclients
	clients, err := d.SPI.Setup(req.Secret)
	if err != nil {
		return nil, err
	}

	// Check if the machine should be assigned to a vnet in a different resource group.
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

	klog.V(3).Infof("Fetching subnet details for VM %q", vmName)

	// Getting the subnet object for subnetName
	subnet, err := clients.GetSubnet().Get(
		ctx,
		vnetResourceGroup,
		vnetName,
		subnetName,
		"",
	)
	if err != nil {
		return nil, OnARMAPIErrorFail(prometheusServiceSubnet, err, "Subnet.Get failed for %s due to %s", subnetName, err)
	}
	OnARMAPISuccess(prometheusServiceSubnet, "subnet.Get")

	NIC, err := clients.GetNic().Get(ctx, resourceGroupName, nicName, "")
	if err != nil {

		if isResourceNotFoundError(err) {

			/*
				NIC creation
			*/

			// Creating NICParameters for new NIC creation request
			NICParameters := getNICParameters(vmName, &subnet, providerSpec)

			// NIC creation request
			klog.V(3).Infof("NIC creation started for %q", nicName)

			NICFuture, err := clients.GetNic().CreateOrUpdate(ctx, resourceGroupName, *NICParameters.Name, NICParameters)
			if err != nil {
				// Since machine creation failed, delete any infra resources created
				deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
				if deleteErr != nil {
					klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
				}

				return nil, OnARMAPIErrorFail(prometheusServiceNIC, err, "NIC.CreateOrUpdate failed for %s", *NICParameters.Name)
			}

			// Wait until NIC is created
			err = NICFuture.WaitForCompletionRef(ctx, clients.GetClient())
			if err != nil {
				// Since machine creation failed, delete any infra resources created
				deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
				if deleteErr != nil {
					klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
				}

				return nil, OnARMAPIErrorFail(prometheusServiceNIC, err, "NIC.WaitForCompletionRef failed for %s", *NICParameters.Name)
			}
			OnARMAPISuccess(prometheusServiceNIC, "NIC.CreateOrUpdate")

			// Fetch NIC details
			NIC, err = NICFuture.Result(clients.GetNicImpl())
			if err != nil {
				// Since machine creation failed, delete any infra resources created
				deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
				if deleteErr != nil {
					klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
				}

				return nil, err
			}
			klog.V(3).Infof("NIC creation was successful for %q", nicName)
		} else {
			// Get on NIC returns a non 404 error. Exiting creation with the error.

			// Since machine creation failed, delete any infra resources created
			deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
			if deleteErr != nil {
				klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
			}

			return nil, OnARMAPIErrorFail(prometheusServiceNIC, err, "NIC.Get failed for %s", nicName)
		}

	} else {
		klog.V(3).Infof("Found existing NIC with matching name, hence adopting NIC with name %q", nicName)
	}

	/*
		VM creation
	*/
	startTime := time.Now()
	imageRefClass := providerSpec.Properties.StorageProfile.ImageReference
	// if ID, shared id and community id are not set the image is referenced using a URN
	if imageRefClass.URN != nil {
		imageReference := getImageReference(providerSpec)
		vmImage, err := clients.GetImages().Get(
			ctx,
			providerSpec.Location,
			*imageReference.Publisher,
			*imageReference.Offer,
			*imageReference.Sku,
			*imageReference.Version)

		if err != nil {
			//Since machine creation failed, delete any infra resources created
			deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
			if deleteErr != nil {
				klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
			}

			return nil, OnARMAPIErrorFail(prometheusServiceVM, err, "VirtualMachineImagesclientutils.Get failed for %s", req.MachineClass.Name)
		}

		if vmImage.Plan != nil {
			// If VMImage.Plan exists, check if agreement is accepted and if not accept it for the subscription

			agreement, err := clients.GetMarketplace().Get(
				ctx,
				*vmImage.Plan.Publisher,
				*vmImage.Plan.Product,
				*vmImage.Plan.Name,
			)

			if err != nil {
				//Since machine creation failed, delete any infra resources created
				deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
				if deleteErr != nil {
					klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
				}

				return nil, OnARMAPIErrorFail(prometheusServiceVM, err, "MarketplaceAgreementsclient.Get failed for %s", req.MachineClass.Name)
			}

			if agreement.Accepted == nil || !*agreement.Accepted {
				// Need to accept the terms at least once for the subscription
				klog.V(2).Info("Accepting terms for subscription to make use of the plan")

				agreement.Accepted = to.BoolPtr(true)
				_, err = clients.GetMarketplace().Create(
					ctx,
					*vmImage.Plan.Publisher,
					*vmImage.Plan.Product,
					*vmImage.Plan.Name,
					agreement,
				)

				if err != nil {
					//Since machine creation failed, delete any infra resources created
					deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
					if deleteErr != nil {
						klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
					}

					return nil, OnARMAPIErrorFail(prometheusServiceVM, err, "MarketplaceAgreementsclientutils.Create failed for %s", req.MachineClass.Name)
				}
			}
		}

		vmImageRef = &vmImage
	}

	// Creating VMParameters for new VM creation request
	VMParameters, err := getVMParameters(vmName, vmImageRef, *NIC.ID, providerSpec, req.Secret)
	if err != nil {
		return nil, err
	}
	// VM creation request
	klog.V(3).Infof("VM creation began for %q", vmName)
	VMFuture, err := clients.GetVM().CreateOrUpdate(ctx, resourceGroupName, *VMParameters.Name, VMParameters)
	if err != nil {
		//Since machine creation failed, delete any infra resources created
		deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, OnARMAPIErrorFail(prometheusServiceVM, err, "GetVM().CreateOrUpdate failed for %s", *VMParameters.Name)
	}
	// Wait until VM is created
	klog.V(3).Infof("Waiting for VM create call completion for %q", vmName)
	err = VMFuture.WaitForCompletionRef(ctx, clients.GetClient())
	if err != nil {
		// Since machine creation failed, delete any infra resources created
		deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, OnARMAPIErrorFail(prometheusServiceVM, err, "VMFuture.WaitForCompletionRef failed for %s", *VMParameters.Name)
	}
	klog.V(3).Infof("VM Created in %d", time.Since(startTime))

	// Fetch VM details
	VM, err := VMFuture.Result(clients.GetVMImpl())
	if err != nil {
		// Since machine creation failed, delete any infra resources created
		deleteErr := d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
		if deleteErr != nil {
			klog.Errorf("Error occurred during resource clean up: %s", deleteErr)
		}

		return nil, OnARMAPIErrorFail(prometheusServiceVM, err, "VMFuture.Result failed for %s", *VMParameters.Name)
	}

	OnARMAPISuccess(prometheusServiceVM, "VM.CreateOrUpdate")
	klog.V(3).Infof("VM has been created succesfully for %q", vmName)

	return &VM, nil
}

// deleteVMNicDisks deletes the VM and associated Disks and NIC
func (d *MachinePlugin) deleteVMNicDisks(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, VMName string, nicName string, diskName string, dataDiskNames []string) error {

	// We try to fetch the VM, detach its data disks and finally delete it
	if vm, vmErr := clients.GetVM().Get(ctx, resourceGroupName, VMName, ""); vmErr == nil {

		if detachmentErr := waitForDataDiskDetachment(ctx, clients, resourceGroupName, vm); detachmentErr != nil {
			return detachmentErr
		}
		if deleteErr := DeleteVM(ctx, clients, resourceGroupName, VMName); deleteErr != nil {
			return deleteErr
		}

		OnARMAPISuccess(prometheusServiceVM, "VM Get was successful for %s", *vm.Name)
	} else if !NotFound(vmErr) {
		// If some other error occurred, which is not 404 Not Found (the VM doesn't exist) then bubble up
		return OnARMAPIErrorFail(prometheusServiceVM, vmErr, "vm.Get")
	}

	// Fetch the NIC and deleted it
	nicDeleter := func() error {
		if vmHoldingNic, err := FetchAttachedVMfromNIC(ctx, clients, resourceGroupName, nicName); err != nil {
			if NotFound(err) {
				// Resource doesn't exist, no need to delete
				return nil
			}
			return err
		} else if vmHoldingNic != "" {
			return fmt.Errorf("cannot delete NIC %s because it is attached to VM %s", nicName, vmHoldingNic)
		}

		return DeleteNIC(ctx, clients, resourceGroupName, nicName)
	}

	// Fetch the system disk and delete it
	diskDeleter := GetDeleterForDisk(ctx, clients, resourceGroupName, diskName)

	deleters := []func() error{nicDeleter, diskDeleter}

	for _, dataDiskName := range dataDiskNames {
		dataDiskDeleter := GetDeleterForDisk(ctx, clients, resourceGroupName, dataDiskName)
		deleters = append(deleters, dataDiskDeleter)
	}

	return RunInParallel(deleters)
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
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	// Extract the Subnet Info object form the AzureMachineClass
	// to fill it up in the MachineClass
	data, _ = json.Marshal(azureMachineClass.Spec.SubnetInfo)
	err = json.Unmarshal(data, &subnetInfo)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

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

	machineClass.Provider = ProviderAzure
	machineClass.SecretRef = azureMachineClass.Spec.SecretRef
	machineClass.CredentialsSecretRef = azureMachineClass.Spec.CredentialsSecretRef
	machineClass.Name = azureMachineClass.Name
	machineClass.Labels = azureMachineClass.Labels
	machineClass.Annotations = azureMachineClass.Annotations
	machineClass.Finalizers = azureMachineClass.Finalizers
	machineClass.ProviderSpec = runtime.RawExtension{
		Raw: providerSpecMarshal,
	}

	return err
}

// WaitForDataDiskDetachment is function that ensures all the data disks are detached from the VM
func waitForDataDiskDetachment(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, vm compute.VirtualMachine) error {
	klog.V(2).Infof("Data disk detachment began for %q", *vm.Name)
	defer klog.V(2).Infof("Data disk detached for %q", *vm.Name)

	if len(*vm.StorageProfile.DataDisks) > 0 {
		// There are disks attached hence need to detach them
		vm.StorageProfile.DataDisks = &[]compute.DataDisk{}

		future, err := clients.GetVM().CreateOrUpdate(ctx, resourceGroupName, *vm.Name, vm)
		if err != nil {
			return OnARMAPIErrorFail(prometheusServiceVM, err, "Failed to CreateOrUpdate. Error Message - %s", err)
		}
		err = future.WaitForCompletionRef(ctx, clients.GetClient())
		if err != nil {
			return OnARMAPIErrorFail(prometheusServiceVM, err, "Failed to CreateOrUpdate. Error Message - %s", err)
		}
		OnARMAPISuccess(prometheusServiceVM, "VM CreateOrUpdate was successful for %s", *vm.Name)
	}

	return nil
}

// FetchAttachedVMfromNIC is a helper function to fetch the attached VM for a particular NIC
func FetchAttachedVMfromNIC(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName, nicName string) (string, error) {
	nic, err := clients.GetNic().Get(ctx, resourceGroupName, nicName, "")
	if err != nil {
		return "", err
	}
	if nic.VirtualMachine == nil {
		return "", nil
	}
	return *nic.VirtualMachine.ID, nil
}

// DeleteNIC function deletes the attached Network Interface Card
func DeleteNIC(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, nicName string) error {

	klog.V(2).Infof("NIC delete started for %q", nicName)
	defer klog.V(2).Infof("NIC deleted for %q", nicName)

	nicDeletionCtx, cancel := context.WithTimeout(ctx, nicDeletionTimeout)
	defer cancel()

	future, err := clients.GetNic().Delete(nicDeletionCtx, resourceGroupName, nicName)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.Delete")
	}

	err = future.WaitForCompletionRef(ctx, clients.GetClient())
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.Delete")
	}

	err = backoff.WaitUntil(
		ctx,
		intialInterval,
		maxInterval,
		maxElapsedTime,
		checkNICStatus(ctx, clients, resourceGroupName, nicName, false),
	)

	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.Delete")
	}

	OnARMAPISuccess(prometheusServiceNIC, "NIC deletion was successful for %s", nicName)
	return nil
}

// DeleteVM is the helper function to acknowledge the VM deletion
func DeleteVM(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, vmName string) error {
	klog.V(2).Infof("VM deletion has began for %q", vmName)
	defer klog.V(2).Infof("VM deleted for %q", vmName)

	forceDeletion := false
	future, err := clients.GetVM().Delete(ctx, resourceGroupName, vmName, &forceDeletion)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceVM, err, "vm.Delete")
	}
	err = future.WaitForCompletionRef(ctx, clients.GetClient())
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceVM, err, "vm.Delete")
	}
	OnARMAPISuccess(prometheusServiceVM, "VM deletion was successful for %s", vmName)
	return nil
}

func fetchAttachedVMfromDisk(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName, diskName string) (string, error) {
	disk, err := clients.GetDisk().Get(ctx, resourceGroupName, diskName)
	if err != nil {
		return "", err
	}
	if disk.ManagedBy == nil {
		return "", nil
	}
	return *disk.ManagedBy, nil
}

func deleteDisk(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, diskName string) error {
	klog.V(2).Infof("Disk delete started for %q", diskName)
	defer klog.V(2).Infof("Disk deleted for %q", diskName)

	future, err := clients.GetDisk().Delete(ctx, resourceGroupName, diskName)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceDisk, err, "disk.Delete")
	}
	if err = future.WaitForCompletionRef(ctx, clients.GetClient()); err != nil {
		return OnARMAPIErrorFail(prometheusServiceDisk, err, "disk.Delete")
	}
	OnARMAPISuccess(prometheusServiceDisk, "Disk deletion was successful for %s", diskName)
	return nil
}

// GetDeleterForDisk executes the deletion of the attached disk
func GetDeleterForDisk(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, diskName string) func() error {
	return func() error {
		if vmHoldingDisk, err := fetchAttachedVMfromDisk(ctx, clients, resourceGroupName, diskName); err != nil {
			if NotFound(err) {
				// Resource doesn't exist, no need to delete
				return nil
			}
			return err
		} else if vmHoldingDisk != "" {
			return fmt.Errorf("cannot delete disk %s because it is attached to VM %s", diskName, vmHoldingDisk)
		}

		return deleteDisk(ctx, clients, resourceGroupName, diskName)
	}
}

// RunInParallel executes multiple functions (which return an error) as go functions concurrently.
func RunInParallel(funcs []func() error) error {
	//
	// Execute multiple functions (which return an error) as go functions concurrently.
	//
	var wg sync.WaitGroup
	wg.Add(len(funcs))

	errors := make([]error, len(funcs))
	for i, funOuter := range funcs {
		go func(results []error, idx int, funInner func() error) {
			defer wg.Done()
			if funInner == nil {
				results[idx] = fmt.Errorf("received nil function")
				return
			}
			err := funInner()
			results[idx] = err
		}(errors, i, funOuter)
	}

	wg.Wait()

	var trimmedErrorMessages []string
	for _, e := range errors {
		if e != nil {
			trimmedErrorMessages = append(trimmedErrorMessages, e.Error())
		}
	}
	if len(trimmedErrorMessages) > 0 {
		return fmt.Errorf(strings.Join(trimmedErrorMessages, "\n"))
	}
	return nil
}

// PrometheusFail ...
func PrometheusFail(service string) {
	metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "azure", "service": service}).Inc()
}

// PrometheusSuccess ..
func PrometheusSuccess(service string) {
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "azure", "service": service}).Inc()
}

// RetrieveRequestID ...
func RetrieveRequestID(err error) (bool, string, *autorest.DetailedError) {
	switch err.(type) {
	case autorest.DetailedError:
		detailedErr := autorest.DetailedError(err.(autorest.DetailedError))
		if detailedErr.Response != nil {
			requestID := strings.Join(detailedErr.Response.Header["X-Ms-Request-Id"], "")
			return true, requestID, &detailedErr
		}
		return false, "", nil
	default:
		return false, "", nil
	}
}

// OnErrorFail prints a failure message and exits the program if err is not nil.
func OnErrorFail(err error, format string, v ...interface{}) error {
	if err != nil {
		message := fmt.Sprintf(format, v...)
		if hasRequestID, requestID, detailedErr := RetrieveRequestID(err); hasRequestID {
			klog.Errorf("Azure ARM API call with x-ms-request-id=%s failed. %s: %s\n", requestID, message, *detailedErr)
		} else {
			klog.Errorf("%s: %s\n", message, err)
		}
	}
	return err
}

// OnARMAPIErrorFail ...
func OnARMAPIErrorFail(prometheusService string, err error, format string, v ...interface{}) error {
	PrometheusFail(prometheusService)
	return OnErrorFail(err, format, v...)
}

// OnARMAPISuccess ...
func OnARMAPISuccess(prometheusService string, format string, v ...interface{}) {
	PrometheusSuccess(prometheusService)
}

// NotFound function return True if the http response error denotes the Not Found Status
func NotFound(err error) bool {
	isDetailedError, _, detailedError := RetrieveRequestID(err)
	return isDetailedError && detailedError.Response.StatusCode == 404
}

func checkNICStatus(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, nicName string, shouldExist bool) func() error {
	return func() error {

		nic, err := clients.GetNic().Get(ctx, resourceGroupName, nicName, "")

		// Case-1: If NIC should exist, check below if condition
		if shouldExist {
			if err == nil && nic.ID != nil {
				// NIC exists
				return nil
			}

			klog.V(4).Infof("NIC %q does not exist", nicName)
			return fmt.Errorf("NIC %q does not exist", nicName)
		}

		// Case-2: If NIC should not exist, check below condition
		if err != nil && isResourceNotFoundError(err) {
			// NIC doesn't exist, hence deletion is successful
			return nil
		}

		klog.V(4).Infof("NIC %q exists", nicName)
		return fmt.Errorf("NIC %q exists", nicName)
	}
}

// isResourceNotFoundError returns true when resource is not found at provider
func isResourceNotFoundError(err error) bool {
	const (
		resourceNotFoundStatusCode = "404"
	)

	if e, ok := err.(autorest.DetailedError); ok {
		statusCode := fmt.Sprintf("%v", e.StatusCode)
		if statusCode == resourceNotFoundStatusCode {
			return true
		}
	}

	return false
}

// getRelevantVMs is a helper method used to list actual vm instances
func getRelevantVMs(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, location string, tags map[string]string) (VMs, error) {
	var (
		listOfVMs         = make(VMs)
		searchClusterName = ""
		searchNodeRole    = ""
	)

	for key := range tags {
		if strings.Contains(key, "kubernetes.io-cluster-") {
			searchClusterName = key
		} else if strings.Contains(key, "kubernetes.io-role-") {
			searchNodeRole = key
		}
	}

	if searchClusterName == "" ||
		searchNodeRole == "" ||
		resourceGroupName == "" {
		return listOfVMs, nil
	}

	machines, err := getAllVMs(ctx, clients, resourceGroupName)
	if err != nil {
		return listOfVMs, err
	}

	if len(machines) > 0 {
		for _, server := range machines {
			if !verifyAzureTags(server.Tags, searchClusterName, searchNodeRole) {
				klog.V(2).Infof("%q VM found, but not verified with tags %s and %s", *server.Name, searchClusterName, searchNodeRole)
				continue
			}

			instanceID := encodeMachineID(location, *server.Name)

			listOfVMs[instanceID] = *server.Name
			klog.V(4).Infof("Found machine with name: %q", *server.Name)

		}
	}

	return listOfVMs, nil
}

// getRelevantNICs is helper method used to list NICs
func getRelevantNICs(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, location string, tags map[string]string) (VMs, error) {
	var (
		listOfVMs         = make(VMs)
		searchClusterName = ""
		searchNodeRole    = ""
	)

	for key := range tags {
		if strings.Contains(key, "kubernetes.io-cluster-") {
			searchClusterName = key
		} else if strings.Contains(key, "kubernetes.io-role-") {
			searchNodeRole = key
		}
	}

	if searchClusterName == "" || searchNodeRole == "" || resourceGroupName == "" {
		return listOfVMs, nil
	}

	interfaces, err := getAllNICs(ctx, clients, resourceGroupName)
	if err != nil {
		return listOfVMs, err
	}

	if len(interfaces) > 0 {
		for _, nic := range interfaces {
			isNic, machineName := vmNameFromDependencyName(*nic.Name, nicSuffix)
			if !isNic {
				continue
			}

			if !verifyAzureTags(nic.Tags, searchClusterName, searchNodeRole) {
				klog.V(2).Infof("%q NIC found, but not verified with tags %s and %s", *nic.Name, searchClusterName, searchNodeRole)
				continue
			}

			instanceID := encodeMachineID(location, machineName)
			listOfVMs[instanceID] = machineName
			klog.V(4).Infof("Found nic with name %q, hence appending machine %q", *nic.Name, machineName)

		}
	}

	return listOfVMs, nil
}

// getRelevantDisks is a helper method used to list disks
func getRelevantDisks(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string, location string, tags map[string]string) (VMs, error) {
	var (
		listOfVMs         = make(VMs)
		searchClusterName = ""
		searchNodeRole    = ""
	)

	for key := range tags {
		if strings.Contains(key, "kubernetes.io-cluster-") {
			searchClusterName = key
		} else if strings.Contains(key, "kubernetes.io-role-") {
			searchNodeRole = key
		}
	}

	if searchClusterName == "" ||
		searchNodeRole == "" ||
		resourceGroupName == "" {
		return listOfVMs, nil
	}

	disks, err := getAllDisks(ctx, clients, resourceGroupName)
	if err != nil {
		return listOfVMs, err
	}

	if len(disks) > 0 {
		for _, disk := range disks {
			if disk.OsType != "" {
				isDisk, machineName := vmNameFromDependencyName(*disk.Name, diskSuffix)
				if !isDisk {
					continue
				}

				if !verifyAzureTags(disk.Tags, searchClusterName, searchNodeRole) {
					klog.V(2).Infof("%q Disk found, but not verified with tags %s and %s", *disk.Name, searchClusterName, searchNodeRole)
					continue
				}

				instanceID := encodeMachineID(location, machineName)

				listOfVMs[instanceID] = machineName
				klog.V(4).Infof("Found disk with name %q, hence appending machine %q", *disk.Name, machineName)
			}
		}
	}

	return listOfVMs, nil
}

func getAllVMs(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string) ([]compute.VirtualMachine, error) {
	var items []compute.VirtualMachine
	result, err := clients.GetVM().List(ctx, resourceGroupName, "")
	if err != nil {
		return items, OnARMAPIErrorFail(prometheusServiceVM, err, "vm.List")
	}
	items = append(items, result.Values()...)
	for result.NotDone() {
		err = result.NextWithContext(ctx)
		if err != nil {
			return items, OnARMAPIErrorFail(prometheusServiceVM, err, "vm.List")
		}
		items = append(items, result.Values()...)
	}
	OnARMAPISuccess(prometheusServiceVM, "vm.List")
	return items, nil
}

func getAllNICs(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string) ([]network.Interface, error) {
	var items []network.Interface
	result, err := clients.GetNic().List(ctx, resourceGroupName)
	if err != nil {
		return items, OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.List")
	}
	items = append(items, result.Values()...)
	for result.NotDone() {
		err = result.NextWithContext(ctx)
		if err != nil {
			return items, OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.List")
		}
		items = append(items, result.Values()...)
	}
	OnARMAPISuccess(prometheusServiceNIC, "nic.List")
	return items, nil
}

func getAllDisks(ctx context.Context, clients spi.AzureDriverClientsInterface, resourceGroupName string) ([]compute.Disk, error) {
	var items []compute.Disk
	result, err := clients.GetDisk().ListByResourceGroup(ctx, resourceGroupName)
	if err != nil {
		return items, OnARMAPIErrorFail(prometheusServiceDisk, err, "disk.ListByResourceGroup")
	}
	items = append(items, result.Values()...)
	for result.NotDone() {
		err = result.NextWithContext(ctx)
		if err != nil {
			return items, OnARMAPIErrorFail(prometheusServiceDisk, err, "disk.ListByResourceGroup")
		}
		items = append(items, result.Values()...)
	}
	OnARMAPISuccess(prometheusServiceDisk, "disk.ListByResourceGroup")
	return items, nil
}

func vmNameFromDependencyName(dependencyName, suffix string) (hasProperSuffix bool, vmName string) {
	if strings.HasSuffix(dependencyName, suffix) {
		hasProperSuffix = true
		vmName = dependencyName[:len(dependencyName)-len(suffix)]
	} else {
		hasProperSuffix = false
		vmName = ""
	}
	return
}

func verifyAzureTags(tags map[string]*string, clusterNameTag, nodeRoleTag string) bool {
	if tags == nil {
		return false
	}

	var clusterNameMatched, nodeRoleMatched bool
	for key := range tags {
		if strings.Contains(key, clusterNameTag) {
			clusterNameMatched = true
		}
		if strings.Contains(key, nodeRoleTag) {
			nodeRoleMatched = true
		}
	}
	if !clusterNameMatched || !nodeRoleMatched {
		return false
	}

	return true
}
