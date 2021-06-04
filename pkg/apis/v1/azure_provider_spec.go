/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package v1 defined the schema of the Azure Provider Spec
package v1

const (
	// AzureClientID is a constant for a key name that is part of the Azure cloud credentials.
	AzureClientID string = "azureClientId"

	// AzureClientSecret is a constant for a key name that is part of the Azure cloud credentials.
	AzureClientSecret string = "azureClientSecret"

	// AzureSubscriptionID is a constant for a key name that is part of the Azure cloud credentials.
	AzureSubscriptionID string = "azureSubscriptionId"

	// AzureTenantID is a constant for a key name that is part of the Azure cloud credentials.
	AzureTenantID string = "azureTenantId"

	// AzureAlternativeClientID is a constant for a key name of a secret containing the Azure credentials (client id).
	AzureAlternativeClientID = "clientID"

	// AzureAlternativeClientSecret is a constant for a key name of a secret containing the Azure credentials (client
	// secret).
	AzureAlternativeClientSecret = "clientSecret"

	// AzureAlternativeSubscriptionID is a constant for a key name of a secret containing the Azure credentials
	// (subscription id).
	AzureAlternativeSubscriptionID = "subscriptionID"

	// AzureAlternativeTenantID is a constant for a key name of a secret containing the Azure credentials (tenant id).
	AzureAlternativeTenantID = "tenantID"

	// MachineSetKindAvailabilitySet is the machine set kind for AvailabilitySet
	MachineSetKindAvailabilitySet string = "availabilityset"

	// MachineSetKindVMO is the machine set kind for VirtualMachineScaleSet Orchestration Mode VM (VMO)
	MachineSetKindVMO string = "vmo"
)

// +genclient

// AzureProviderSpec is the provider specific configuration to use during node creation
// on Azure.
type AzureProviderSpec struct {

	// Region in which virtual machine would be hosted.
	Location string `json:"location,omitempty"`

	// Identifier tags for virtual machines.
	Tags map[string]string `json:"tags,omitempty"`

	// AzureVirtualMachineProperties describes the properties of a Virtual Machine.
	Properties AzureVirtualMachineProperties `json:"properties,omitempty"`

	// Name of the Azure resource group
	ResourceGroup string `json:"resourceGroup,omitempty"`

	// AzureSubnetInfo is the information containing the subnet details
	SubnetInfo AzureSubnetInfo `json:"subnetInfo,omitempty"`
}

// AzureVirtualMachineProperties describes the properties of a Virtual Machine.
type AzureVirtualMachineProperties struct {
	// Specifies the hardware settings for the virtual machine.
	HardwareProfile AzureHardwareProfile `json:"hardwareProfile,omitempty"`

	// Specifies the storage settings for the virtual machine disks.
	StorageProfile AzureStorageProfile `json:"storageProfile,omitempty"`

	// Specifies the operating system settings used while creating the virtual
	// machine. Some of the settings cannot be changed once VM is provisioned.
	OsProfile AzureOSProfile `json:"osProfile,omitempty"`

	// Specifies the network interfaces of the virtual machine.
	NetworkProfile AzureNetworkProfile `json:"networkProfile,omitempty"`

	// Specifies information about the availability set that the virtual
	// machine should be assigned to. Virtual machines specified in the
	// same availability set are allocated to different nodes to maximize
	// availability. For more information about availability sets, see
	// Manage the availability of virtual machines.
	//
	// Currently, a VM can only be added to availability set at creation
	// time. The availability set to which the VM is being added should
	// be under the same resource group as the availability set resource.
	// An existing VM cannot be added to an availability set.
	AvailabilitySet *AzureSubResource `json:"availabilitySet,omitempty"`

	// The identity of the virtual machine.
	IdentityID *string `json:"identityID,omitempty"`

	// The virtual machine zone.
	Zone *int `json:"zone,omitempty"`

	// AzureMachineSetConfig contains the information about the associated machineSet.
	MachineSet *AzureMachineSetConfig `json:"machineSet,omitempty"`
}

// AzureHardwareProfile is specifies the hardware settings for the virtual machine.
// Refer github.com/Azure/azure-sdk-for-go/arm/compute/models.go for VMSizes
type AzureHardwareProfile struct {
	// Specifies the size of the virtual machine. The enum data type is currently
	// deprecated and will be removed by December 23rd 2023. Recommended way to get
	// the list of available sizes is using these APIs:
	//
	// - List all available virtual machine sizes in an availability set
	// - List all available virtual machine sizes in a region
	// - List all available virtual machine sizes for resizing.
	//
	// The available VM sizes depend on region and availability set.
	VMSize string `json:"vmSize,omitempty"`
}

// AzureMachineSetConfig contains the information about the associated machineSet.
type AzureMachineSetConfig struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// AzureStorageProfile is specifies the storage settings for the virtual machine disks.
type AzureStorageProfile struct {

	// Specifies information about the image to use. You can specify information about platform
	// images, marketplace images, or virtual machine images. This element is required when you want
	// to use a platform image, marketplace image, or virtual machine image, but is not used in other
	// creation operations.
	ImageReference AzureImageReference `json:"imageReference,omitempty"`

	// Specifies information about the operating system disk used by the virtual machine.
	// For more information about disks, see [About disks and VHDs for Azure virtual
	// machines](https://docs.microsoft.com/en-us/azure/virtual-machines/managed-disks-overview).
	OsDisk AzureOSDisk `json:"osDisk,omitempty"`

	// Specifies the parameters that are used to add a data disk to a virtual machine.
	// For more information about disks, see [About disks and VHDs for Azure virtual
	// machines](https://docs.microsoft.com/en-us/azure/virtual-machines/managed-disks-overview).
	DataDisks []AzureDataDisk `json:"dataDisks,omitempty"`
}

// AzureImageReference specifies information about the image to use. You can specify information about platform images,
// marketplace images, or virtual machine images. This element is required when you want to use a platform image,
// marketplace image, or virtual machine image, but is not used in other creation operations.
type AzureImageReference struct {
	// Resource Id
	ID string `json:"id,omitempty"`

	// Uniform Resource Name of the OS image to be used , it has the format 'publisher:offer:sku:version'
	URN *string `json:"urn,omitempty"`
}

// AzureOSDisk specifies information about the operating system disk used by the virtual machine.
// For more information about disks, see [About disks and VHDs for Azure virtual
// machines](https://docs.microsoft.com/azure/virtual-machines/virtual-machines-windows-about-disks-vhds?toc=%2fazure%2fvirtual-machines%2fwindows%2ftoc.json).
type AzureOSDisk struct {
	// The disk name.
	Name string `json:"name,omitempty"`

	// Specifies the caching requirements.
	// Possible values are: *None*, *ReadOnly* or *ReadWrite*
	// Default: *None* for Standard storage. *ReadOnly* for Premium storage.
	Caching string `json:"caching,omitempty"`

	// The managed disk parameters.
	ManagedDisk AzureManagedDiskParameters `json:"managedDisk,omitempty"`

	// Specifies the size of an empty data disk in gigabytes. This element can be used to
	// overwrite the size of the disk in a virtual machine image.
	//
	// This value cannot be larger than 1023 GB.
	DiskSizeGB int32 `json:"diskSizeGB,omitempty"`

	// Specifies how the virtual machine should be created.
	//
	// Possible values are:
	//
	// **Attach** \u2013 This value is used when you are using a specialized disk to create the virtual machine.
	//
	// **FromImage** \u2013 This value is used when you are using an image to create the virtual machine. If you
	// are using a platform image, you also use the imageReference element described above. If you are using a
	// marketplace image, you also use the plan element previously described.
	CreateOption string `json:"createOption,omitempty"`
}

// AzureDataDisk Specifies the parameters that are used to add a data disk to a virtual machine.
// For more information about disks, see [About disks and VHDs for Azure virtual machines](https://docs.microsoft.com/en-us/azure/virtual-machines/managed-disks-overview).
type AzureDataDisk struct {
	// The disk name.
	Name string `json:"name,omitempty"`

	// Specifies the logical unit number of the data disk. This value is used to identify data disks within the VM
	// and therefore must be unique for each data disk attached to a VM.
	Lun *int32 `json:"lun,omitempty"`

	// Specifies the caching requirements.
	//
	// Possible values are: *None*, *ReadOnly*, *ReadWrite*
	//
	// Default: *None* for Standard storage. *ReadOnly* for Premium storage
	Caching string `json:"caching,omitempty"`

	// Specifies the storage account type for the managed disk.
	// NOTE: UltraSSD_LRS can only be used with data disks, it cannot be used with OS Disk.
	StorageAccountType string `json:"storageAccountType,omitempty"`

	// Specifies the size of an empty data disk in gigabytes. This element can be used to overwrite the size of the disk in a virtual machine image.
	//
	// This value cannot be larger than 1023 GB
	DiskSizeGB int32 `json:"diskSizeGB,omitempty"`
}

// AzureManagedDiskParameters is the parameters of a managed disk.
type AzureManagedDiskParameters struct {
	// Resource Id
	ID string `json:"id,omitempty"`

	// Specifies the storage account type for the managed disk.
	// NOTE: UltraSSD_LRS can only be used with data disks, it cannot be used with OS Disk.
	StorageAccountType string `json:"storageAccountType,omitempty"`
}

// AzureOSProfile specifies the operating system settings for the virtual machine.
// Some of the settings cannot be changed once VM is provisioned. For more details
// see [documentation on osProfile in Azure Virtual Machines](https://docs.microsoft.com/en-us/rest/api/compute/virtual-machines/create-or-update#osprofile)
type AzureOSProfile struct {

	// Specifies the host OS name of the virtual machine. This name cannot be updated after the VM is created.
	//
	// **Max-length (Windows)**: 15 characters
	//
	// **Max-length (Linux)**: 64 characters.
	//
	// For naming conventions and restrictions see [Azure infrastructure services implementation
	// guidelines](https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules).
	ComputerName string `json:"computerName,omitempty"`

	// Specifies the name of the administrator account.
	// This property cannot be updated after the VM is created.

	// **Windows-only restriction**: Cannot end in "."
	//
	// **Disallowed values**: "administrator", "admin", "user", "user1", "test", "user2", "test1", "user3", "admin1",
	// "1", "123", "a", "actuser", "adm", "admin2", "aspnet", "backup", "console", "david", "guest", "john", "owner",
	// "root", "server", "sql", "support", "support_388945a0", "sys", "test2", "test3", "user4", "user5".
	//
	// **Minimum-length (Linux)**: 1 character
	//
	// **Max-length (Linux)**: 64 characters
	//
	// **Max-length (Windows)**: 20 characters.
	AdminUsername string `json:"adminUsername,omitempty"`

	// Specifies the password of the administrator account.
	//
	// **Minimum-length (Windows)**: 8 characters
	//
	// **Minimum-length (Linux)**: 6 characters
	//
	// **Max-length (Windows)**: 123 characters
	//
	// **Max-length (Linux)**: 72 characters
	//
	// **Complexity requirements**: 3 out of 4 conditions below need to be fulfilled
	// Has lower characters
	// Has upper characters
	// Has a digit
	// Has a special character (Regex match [\W_])
	//
	// **Disallowed values**: "abc@123", "P@$$w0rd", "P@ssw0rd", "P@ssword123", "Pa$$word", "pass@word1",
	// "Password!", "Password1", "Password22", "iloveyou!"
	//
	// For resetting the password, see [How to reset the Remote Desktop service or its login password in a Windows
	// VM](https://docs.microsoft.com/en-us/troubleshoot/azure/virtual-machines/reset-rdp)
	//
	// For resetting root password, see [Manage users, SSH, and check or repair disks on Azure Linux VMs using the
	// VMAccess Extension](https://docs.microsoft.com/en-us/troubleshoot/azure/virtual-machines/troubleshoot-ssh-connection)
	AdminPassword string `json:"adminPassword,omitempty"`

	// 	Specifies a base-64 encoded string of custom data. The base-64 encoded string is decoded to a binary array that is saved as a file on the Virtual Machine. The maximum length of the binary array is 65535 bytes.

	// **Note: Do not pass any secrets or passwords in customData property**

	// This property cannot be updated after the VM is created.
	//
	// customData is passed to the VM to be saved as a file, for more information see [Custom Data on Azure
	// VMs](https://azure.microsoft.com/blog/custom-data-and-cloud-init-on-windows-azure/)
	//
	// For using cloud-init for your Linux VM, see [Using cloud-init to customize a Linux VM during
	// creation](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/using-cloud-init)
	CustomData string `json:"customData,omitempty"`

	// 	Specifies the Linux operating system settings on the virtual machine.

	// For a list of supported Linux distributions, see [Linux on Azure-Endorsed
	// Distributions](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/endorsed-distros).
	LinuxConfiguration AzureLinuxConfiguration `json:"linuxConfiguration,omitempty"`
}

// AzureLinuxConfiguration is specifies the Linux operating system settings on the virtual machine.
//
// For a list of supported Linux distributions, see [Linux on Azure-Endorsed
// Distributions](https://docs.microsoft.com/azure/virtual-machines/virtual-machines-linux-endorsed-distros?toc=%2fazure%2fvirtual-machines%2flinux%2ftoc.json)
//
// For running non-endorsed distributions, see [Information for Non-Endorsed
// Distributions](https://docs.microsoft.com/azure/virtual-machines/virtual-machines-linux-create-upload-generic?toc=%2fazure%2fvirtual-machines%2flinux%2ftoc.json).
type AzureLinuxConfiguration struct {

	// Specifies whether password authentication should be disabled.
	DisablePasswordAuthentication bool `json:"disablePasswordAuthentication,omitempty"`

	// Specifies the ssh key configuration for a Linux OS.
	SSH AzureSSHConfiguration `json:"ssh,omitempty"`
}

// AzureSSHConfiguration specifies the ssh key configuration for a Linux OS.
type AzureSSHConfiguration struct {

	// The list of SSH public keys used to authenticate with linux based VMs.
	PublicKeys AzureSSHPublicKey `json:"publicKeys,omitempty"`
}

// AzureSSHPublicKey the list of SSH public keys used to authenticate with linux based VMs.
// key is placed.
type AzureSSHPublicKey struct {

	// Specifies the full path on the created VM where ssh public key is stored. If the file already exists, the specified key is appended to the file. Example: /home/user/.ssh/authorized_keys
	Path string `json:"path,omitempty"`

	// SSH public key certificate used to authenticate with the VM through ssh. The key needs to be at least 2048-bit and in ssh-rsa format.
	//
	// For creating ssh keys, see [Create SSH keys on Linux and Mac for Linux VMs in Azure]https://docs.microsoft.com/azure/virtual-machines/linux/create-ssh-keys-detailed).
	KeyData string `json:"keyData,omitempty"`
}

// AzureNetworkProfile specifies the network interfaces of the virtual machine.
type AzureNetworkProfile struct {
	// Specifies the network interfaces of the virtual machine.
	NetworkInterfaces AzureNetworkInterfaceReference `json:"networkInterfaces,omitempty"`

	// Specifies if the acceleration is enabled in network.
	AcceleratedNetworking *bool `json:"acceleratedNetworking,omitempty"`
}

// AzureNetworkInterfaceReference specifies the network interfaces of the virtual machine.
type AzureNetworkInterfaceReference struct {
	// Resource Id
	ID string `json:"id,omitempty"`

	// Specifies the primary network interface in case the virtual machine has
	// more than 1 network interface.
	*AzureNetworkInterfaceReferenceProperties `json:"properties,omitempty"`
}

// AzureNetworkInterfaceReferenceProperties is describes a network interface
// reference properties.
type AzureNetworkInterfaceReferenceProperties struct {
	// Specifies the primary network interface in case the virtual machine
	// has more than 1 network interface.
	Primary bool `json:"primary,omitempty"`
}

// AzureSubResource specifies information about the availability set that the virtual machine
// should be assigned to. Virtual machines specified in the same availability set
// are allocated to different nodes to maximize availability. For more information
// about availability sets, see Manage the availability of virtual machines.
//
// Currently, a VM can only be added to availability set at creation time.
// The availability set to which the VM is being added should be under the
// same resource group as the availability set resource. An existing VM cannot
//  be added to an availability set.
type AzureSubResource struct {
	// This denotes the resource ID.
	ID string `json:"id,omitempty"`
}

// AzureSubnetInfo is the information containing the subnet details
type AzureSubnetInfo struct {
	// The vNet Name.
	VnetName string `json:"vnetName,omitempty"`

	// The resource group of the vNet.
	VnetResourceGroup *string `json:"vnetResourceGroup,omitempty"`

	// The name of the Subnet that will be utilised by the VM.
	SubnetName string `json:"subnetName,omitempty"`
}
