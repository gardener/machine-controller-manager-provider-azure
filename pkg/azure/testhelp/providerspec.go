// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testhelp

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

// ProviderSpecBuilder is a builder for ProviderSpec. Only used for unit tests.
type ProviderSpecBuilder struct {
	shootNs        string
	workerPoolName string
	spec           api.AzureProviderSpec
}

// NewProviderSpecBuilder creates a new instance of ProviderSpecBuilder.
func NewProviderSpecBuilder(resourceGroup, shootNs, workerPoolName string) *ProviderSpecBuilder {
	return &ProviderSpecBuilder{
		shootNs:        shootNs,
		workerPoolName: workerPoolName,
		spec: api.AzureProviderSpec{
			Location:      Location,
			ResourceGroup: resourceGroup,
			Properties: api.AzureVirtualMachineProperties{
				Zone:            to.Ptr(1),
				StorageProfile:  api.AzureStorageProfile{},
				OsProfile:       api.AzureOSProfile{},
				NetworkProfile:  api.AzureNetworkProfile{},
				HardwareProfile: api.AzureHardwareProfile{},
			},
		},
	}
}

// WithDefaultValues initializes ProviderSpecBuilder with default values for all mandatory fields.
// This sets up sufficient fields so that ProviderSpec validation succeeds.
// NOTE: In case validation is changed this method should adapt.
func (b *ProviderSpecBuilder) WithDefaultValues() *ProviderSpecBuilder {
	return b.
		WithDefaultTags().
		WithDefaultStorageProfile().
		WithDefaultHardwareProfile().
		WithDefaultOsProfile().
		WithDefaultSubnetInfo().
		WithDefaultNetworkProfile()
}

// WithDefaultSubnetInfo sets a default SubnetInfo to the provider spec.
func (b *ProviderSpecBuilder) WithDefaultSubnetInfo() *ProviderSpecBuilder {
	b.spec.SubnetInfo = api.AzureSubnetInfo{
		VnetName:   b.shootNs,
		SubnetName: fmt.Sprintf("%s-nodes", b.shootNs),
	}
	return b
}

// WithSubnetInfo sets a custom vnet resource group to the SubnetInfo part of the provider spec.
func (b *ProviderSpecBuilder) WithSubnetInfo(vnetResourceGroup string) *ProviderSpecBuilder {
	b.spec.SubnetInfo = api.AzureSubnetInfo{
		VnetName:          b.shootNs,
		VnetResourceGroup: to.Ptr(vnetResourceGroup),
		SubnetName:        fmt.Sprintf("%s-nodes", b.shootNs),
	}
	return b
}

// WithDefaultNetworkProfile sets a default network profile in the provider spec.
func (b *ProviderSpecBuilder) WithDefaultNetworkProfile() *ProviderSpecBuilder {
	b.spec.Properties.NetworkProfile = api.AzureNetworkProfile{
		AcceleratedNetworking: to.Ptr(true),
	}
	return b
}

// WithDefaultHardwareProfile sets a default hardware profile in the provider spec.
func (b *ProviderSpecBuilder) WithDefaultHardwareProfile() *ProviderSpecBuilder {
	b.spec.Properties.HardwareProfile.VMSize = VMSize
	return b
}

// WithStorageProfile sets a default storage profile in the provider spec.
func (b *ProviderSpecBuilder) WithStorageProfile(skipMarketplaceAgreement bool, securityEncryption *string) *ProviderSpecBuilder {
	b.spec.Properties.StorageProfile.ImageReference = api.AzureImageReference{
		URN:                      to.Ptr(DefaultImageRefURN),
		SkipMarketplaceAgreement: skipMarketplaceAgreement,
	}
	b.spec.Properties.StorageProfile.OsDisk = api.AzureOSDisk{
		Caching: "None",
		ManagedDisk: api.AzureManagedDiskParameters{
			StorageAccountType: StorageAccountType,
			SecurityProfile: &api.AzureDiskSecurityProfile{
				SecurityEncryptionType: securityEncryption,
			},
		},
		DiskSizeGB:   50,
		CreateOption: "FromImage",
	}
	return b
}

// WithDefaultStorageProfile sets a default storage profile in the provider spec.
func (b *ProviderSpecBuilder) WithDefaultStorageProfile() *ProviderSpecBuilder {
	b.spec.Properties.StorageProfile.ImageReference = api.AzureImageReference{
		URN: to.Ptr(DefaultImageRefURN),
	}
	b.spec.Properties.StorageProfile.OsDisk = api.AzureOSDisk{
		Caching: "None",
		ManagedDisk: api.AzureManagedDiskParameters{
			StorageAccountType: StorageAccountType,
		},
		DiskSizeGB:   50,
		CreateOption: "FromImage",
	}
	return b
}

// WithDataDisks configures data disks in the provider spec.
func (b *ProviderSpecBuilder) WithDataDisks(diskName string, numDisks int) *ProviderSpecBuilder {
	dataDisks := make([]api.AzureDataDisk, 0, numDisks)
	for i := 0; i < numDisks; i++ {
		d := api.AzureDataDisk{
			Name:               diskName,
			Lun:                int32(i),
			Caching:            "None",
			StorageAccountType: StorageAccountType,
			DiskSizeGB:         20,
		}
		dataDisks = append(dataDisks, d)
	}
	b.spec.Properties.StorageProfile.DataDisks = dataDisks
	return b
}

// WithSecurityProfile configures the security profile for the VM.
func (b *ProviderSpecBuilder) WithSecurityProfile(sec *api.AzureSecurityProfile) *ProviderSpecBuilder {
	b.spec.Properties.SecurityProfile = sec
	return b
}

// WithDefaultOsProfile sets a default OS profile in the provider spec.
func (b *ProviderSpecBuilder) WithDefaultOsProfile() *ProviderSpecBuilder {
	b.spec.Properties.OsProfile = api.AzureOSProfile{
		AdminUsername:      "core",
		LinuxConfiguration: api.AzureLinuxConfiguration{},
	}
	return b
}

// WithDefaultTags sets default tags in the provider spec.
func (b *ProviderSpecBuilder) WithDefaultTags() *ProviderSpecBuilder {
	if b.spec.Tags == nil {
		b.spec.Tags = make(map[string]string)
	}
	b.spec.Tags["Name"] = b.shootNs
	b.spec.Tags["kubernetes.io-cluster-"+b.shootNs] = "1"
	b.spec.Tags["kubernetes.io-role-node"] = "1"
	b.spec.Tags["node.kubernetes.io_role"] = "node"
	b.spec.Tags["worker.gardener.cloud_pool"] = b.workerPoolName
	b.spec.Tags["worker.garden.sapcloud.io_group"] = b.workerPoolName
	b.spec.Tags["worker.gardener.cloud_cri-name"] = "containerd"
	b.spec.Tags["worker.gardener.cloud_system-components"] = "true"
	b.spec.Tags["networking.gardener.cloud_node-local-dns-enabled"] = "true"

	return b
}

// WithTags sets custom tags in the provider spec.
func (b *ProviderSpecBuilder) WithTags(tags map[string]string) *ProviderSpecBuilder {
	b.spec.Tags = tags
	return b
}

// Marshal serializes the provider spec to a slice of bytes.
func (b *ProviderSpecBuilder) Marshal() ([]byte, error) {
	return json.Marshal(b.spec)
}

// Build builds the provider spec.
func (b *ProviderSpecBuilder) Build() api.AzureProviderSpec {
	return b.spec
}

// CreateDataDiskNames creates data disk names for the given vm name and provider spec.
func CreateDataDiskNames(vmName string, spec api.AzureProviderSpec) []string {
	var diskNames []string
	for _, specDataDisk := range spec.Properties.StorageProfile.DataDisks {
		diskNames = append(diskNames, utils.CreateDataDiskName(vmName, specDataDisk))
	}
	return diskNames
}
