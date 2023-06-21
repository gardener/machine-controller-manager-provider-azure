/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package api defined the schema of the Azure Provider Spec
package api

const (
	// AzureClientID is a constant for a key name that is part of the Azure cloud credentials.
	// Deprecated: Use ClientID instead.
	AzureClientID string = "azureClientId"
	// AzureClientSecret is a constant for a key name that is part of the Azure cloud credentials.
	// Deprecated: Use ClientSecret instead
	AzureClientSecret string = "azureClientSecret"
	// AzureSubscriptionID is a constant for a key name that is part of the Azure cloud credentials.
	// Deprecated: Use SubscriptionID instead
	AzureSubscriptionID string = "azureSubscriptionId"
	// AzureTenantID is a constant for a key name that is part of the Azure cloud credentials.
	// Deprecated: Use TenantID instead
	AzureTenantID string = "azureTenantId"

	// AzureAlternativeClientID is a constant for a key name of a secret containing the Azure credentials (client id).
	// Deprecated: Use ClientID instead.
	AzureAlternativeClientID = "clientID"
	// AzureAlternativeClientSecret is a constant for a key name of a secret containing the Azure credentials (client
	// secret).
	// Deprecated: Use ClientSecret instead
	AzureAlternativeClientSecret = "clientSecret"
	// AzureAlternativeSubscriptionID is a constant for a key name of a secret containing the Azure credentials
	// (subscription id).
	// Deprecated: Use ClientID instead.
	AzureAlternativeSubscriptionID = "subscriptionID"
	// AzureAlternativeTenantID is a constant for a key name of a secret containing the Azure credentials (tenant id).
	// Deprecated: Use TenantID instead
	AzureAlternativeTenantID = "tenantID"

	// ClientID is a constant for a key name that is part of the Azure cloud credentials.
	ClientID string = "clientID"
	// ClientSecret is a constant for a key name that is part of the Azure cloud credentials.
	ClientSecret string = "clientSecret"
	// SubscriptionID is a constant for a key name that is part of the Azure cloud credentials.
	SubscriptionID string = "subscriptionID"
	// TenantID is a constant for a key name that is part of the Azure cloud credentials.
	TenantID string = "tenantID"

	// MachineSetKindAvailabilitySet is the machine set kind for AvailabilitySet.
	// Deprecated. Use AzureVirtualMachineProperties.AvailabilitySet instead.
	MachineSetKindAvailabilitySet string = "availabilityset"
	// MachineSetKindVMO is the machine set kind for VirtualMachineScaleSet Orchestration Mode VM (VMO).
	// Deprecated. Use AzureVirtualMachineProperties.VirtualMachineScaleSet instead.
	MachineSetKindVMO string = "vmo"
)

// AzureProviderSpec is the spec to be used while parsing the calls.
type AzureProviderSpec struct {
	Location      string                        `json:"location,omitempty"`
	Tags          map[string]string             `json:"tags,omitempty"`
	Properties    AzureVirtualMachineProperties `json:"properties,omitempty"`
	ResourceGroup string                        `json:"resourceGroup,omitempty"`
	SubnetInfo    AzureSubnetInfo               `json:"subnetInfo,omitempty"`
}

// AzureVirtualMachineProperties describes the properties of a Virtual Machine.
type AzureVirtualMachineProperties struct {
	// HardwareProfile specifies the hardware settings for the virtual machine. Currently only VMSize is supported.
	HardwareProfile AzureHardwareProfile `json:"hardwareProfile,omitempty"`
	// StorageProfile specifies the storage settings for the virtual machine.
	StorageProfile AzureStorageProfile `json:"storageProfile,omitempty"`
	// OsProfile specifies the operating system settings used when the virtual machine is created.
	OsProfile AzureOSProfile `json:"osProfile,omitempty"`
	// NetworkProfile specifies the network interfaces for the virtual machine.
	NetworkProfile AzureNetworkProfile `json:"networkProfile,omitempty"`
	// AvailabilitySet specifies the availability set to be associated with the virtual machine.
	// For additional information see: [https://learn.microsoft.com/en-us/azure/virtual-machines/availability-set-overview]
	// Points to note:
	// 1. A VM can only be added to availability set at creation time.
	// 2. The availability set to which the VM is being added should be under the same resource group as the availability set resource.
	// 3. Either of AvailabilitySet or VirtualMachineScaleSet should be specified but not both.
	AvailabilitySet *AzureSubResource `json:"availabilitySet,omitempty"`
	// IdentityID is the managed identity that is associated to the virtual machine.
	// NOTE: Currently only user assigned managed identity is supported.
	// For additional information see the following links:
	// 1. [https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview]
	// 2: [https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/qs-configure-portal-windows-vm]
	IdentityID *string `json:"identityID,omitempty"`
	// Zone is an availability zone where the virtual machine will be created.
	Zone *int `json:"zone,omitempty"`
	// VirtualMachineScaleSet specifies the virtual machine scale set to be associated with the virtual machine.
	// For additional information see: [https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/]
	// Points to note:
	// 1. A VM can only be added to availability set at creation time.
	// 2. Either of AvailabilitySet or VirtualMachineScaleSet should be specified but not both.
	VirtualMachineScaleSet *AzureSubResource `json:"virtualMachineScaleSet,omitempty"`
	// Deprecated. Use either AvailabilitySet or VirtualMachineScaleSet instead
	MachineSet *AzureMachineSetConfig `json:"machineSet,omitempty"`
}

// AzureHardwareProfile specifies the hardware settings for the virtual machine.
// Refer to the [azure-sdk-for-go repository](https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/resourcemanager/compute/armcompute/models.go) for VMSizes.
type AzureHardwareProfile struct {
	// VMSize is an alias for different machine sizes supported by the provider.
	// See [https://docs.microsoft.com/azure/virtual-machines/sizes].The available VM sizes depend on region and availability set.
	VMSize string `json:"vmSize,omitempty"`
}

// AzureMachineSetConfig contains the information about the machine set.
// Deprecated. This type should not be used to differentiate between VirtualMachineScaleSet and AvailabilitySet as
// there are now dedicated struct fields for these.
type AzureMachineSetConfig struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// AzureStorageProfile specifies the storage settings for the virtual machine disks.
type AzureStorageProfile struct {
	ImageReference AzureImageReference `json:"imageReference,omitempty"`
	OsDisk         AzureOSDisk         `json:"osDisk,omitempty"`
	DataDisks      []AzureDataDisk     `json:"dataDisks,omitempty"`
}

// AzureImageReference specifies information about the image to use. You can specify information about platform images,
// marketplace images, community images, shared gallery images or virtual machine images. This element is required when you want to use a platform image,
// marketplace image, community image, shared gallery image or virtual machine image, but is not used in other creation operations.
type AzureImageReference struct {
	ID string `json:"id,omitempty"`
	// Uniform Resource Name of the OS image to be used, it has the format 'publisher:offer:sku:version'
	URN *string `json:"urn,omitempty"`
	// CommunityGalleryImageID is the id of the OS image to be used, hosted within an Azure Community Image Gallery.
	CommunityGalleryImageID *string `json:"communityGalleryImageID,omitempty"`
	// SharedGalleryImageID is the id of the OS image to be used, hosted within an Azure Shared Image Gallery.
	SharedGalleryImageID *string `json:"sharedGalleryImageID,omitempty"`
}

// AzureOSDisk specifies information about the operating system disk used by the virtual machine. <br><br> For more
// information about disks, see [Introduction to Azure Managed
// Disks](https://learn.microsoft.com/en-us/azure/virtual-machines/managed-disks-overview).
type AzureOSDisk struct {
	Name         string                     `json:"name,omitempty"`
	Caching      string                     `json:"caching,omitempty"`
	ManagedDisk  AzureManagedDiskParameters `json:"managedDisk,omitempty"`
	DiskSizeGB   int32                      `json:"diskSizeGB,omitempty"`
	CreateOption string                     `json:"createOption,omitempty"`
}

// AzureDataDisk specifies information about the data disk used by the virtual machine.
type AzureDataDisk struct {
	Name               string `json:"name,omitempty"`
	Lun                *int32 `json:"lun,omitempty"`
	Caching            string `json:"caching,omitempty"`
	StorageAccountType string `json:"storageAccountType,omitempty"`
	DiskSizeGB         int32  `json:"diskSizeGB,omitempty"`
}

// AzureManagedDiskParameters is the parameters of a managed disk.
type AzureManagedDiskParameters struct {
	ID                 string `json:"id,omitempty"`
	StorageAccountType string `json:"storageAccountType,omitempty"`
}

// AzureOSProfile specifies the operating system settings for the virtual machine.
type AzureOSProfile struct {
	ComputerName       string                  `json:"computerName,omitempty"`
	AdminUsername      string                  `json:"adminUsername,omitempty"`
	AdminPassword      string                  `json:"adminPassword,omitempty"`
	CustomData         string                  `json:"customData,omitempty"`
	LinuxConfiguration AzureLinuxConfiguration `json:"linuxConfiguration,omitempty"`
}

// AzureLinuxConfiguration specifies the Linux operating system settings on the virtual machine. <br><br>For a list of
// supported Linux distributions, see [Linux on Azure-Endorsed
// Distributions](https://learn.microsoft.com/en-us/azure/virtual-machines/linux/endorsed-distros).
type AzureLinuxConfiguration struct {
	DisablePasswordAuthentication bool                  `json:"disablePasswordAuthentication,omitempty"`
	SSH                           AzureSSHConfiguration `json:"ssh,omitempty"`
}

// AzureSSHConfiguration is SSH configuration for Linux based VMs running on Azure.
type AzureSSHConfiguration struct {
	PublicKeys AzureSSHPublicKey `json:"publicKeys,omitempty"`
}

// AzureSSHPublicKey is contains information about SSH certificate public key and the path on the Linux VM where the public
// key is placed.
type AzureSSHPublicKey struct {
	Path    string `json:"path,omitempty"`
	KeyData string `json:"keyData,omitempty"`
}

// AzureNetworkProfile specifies the network interfaces of the virtual machine.
type AzureNetworkProfile struct {
	NetworkInterfaces     AzureNetworkInterfaceReference `json:"networkInterfaces,omitempty"`
	AcceleratedNetworking *bool                          `json:"acceleratedNetworking,omitempty"`
}

// AzureNetworkInterfaceReference describes a network interface reference.
type AzureNetworkInterfaceReference struct {
	ID                                        string `json:"id,omitempty"`
	*AzureNetworkInterfaceReferenceProperties `json:"properties,omitempty"`
}

// AzureNetworkInterfaceReferenceProperties describes a network interface reference properties.
type AzureNetworkInterfaceReferenceProperties struct {
	Primary bool `json:"primary,omitempty"`
}

// AzureSubResource is the Sub Resource definition.
type AzureSubResource struct {
	ID string `json:"id,omitempty"`
}

// AzureSubnetInfo is the information containing the subnet details.
type AzureSubnetInfo struct {
	VnetName          string  `json:"vnetName,omitempty"`
	VnetResourceGroup *string `json:"vnetResourceGroup,omitempty"`
	SubnetName        string  `json:"subnetName,omitempty"`
}
