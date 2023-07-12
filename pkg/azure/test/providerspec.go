package test

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"k8s.io/utils/pointer"
)

type ProviderSpecBuilder struct {
	shootNs        string
	workerPoolName string
	spec           api.AzureProviderSpec
}

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

func (b *ProviderSpecBuilder) WithDefaultValues() *ProviderSpecBuilder {
	return b.
		WithDefaultTags().
		WithDefaultStorageProfile().
		WithDefaultHardwareProfile().
		WithDefaultOsProfile().
		WithDefaultSubnetInfo()
}

func (b *ProviderSpecBuilder) WithDefaultSubnetInfo() *ProviderSpecBuilder {
	b.spec.SubnetInfo = api.AzureSubnetInfo{
		VnetName:   b.shootNs,
		SubnetName: fmt.Sprintf("%s-nodes", b.shootNs),
	}
	return b
}

func (b *ProviderSpecBuilder) WithSubnetInfo(vnetResourceGroup string) *ProviderSpecBuilder {
	b.spec.SubnetInfo = api.AzureSubnetInfo{
		VnetName:          b.shootNs,
		VnetResourceGroup: to.Ptr(vnetResourceGroup),
		SubnetName:        fmt.Sprintf("%s-nodes", b.shootNs),
	}
	return b
}

func (b *ProviderSpecBuilder) WithDefaultHardwareProfile() *ProviderSpecBuilder {
	b.spec.Properties.HardwareProfile.VMSize = VMSize
	return b
}

func (b *ProviderSpecBuilder) WithDefaultStorageProfile() *ProviderSpecBuilder {
	b.spec.Properties.StorageProfile.ImageReference = api.AzureImageReference{
		URN: to.Ptr(ImageRefURN),
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

func (b *ProviderSpecBuilder) WithDataDisks(diskNames []string) *ProviderSpecBuilder {
	dataDisks := make([]api.AzureDataDisk, 0, len(diskNames))
	var lun int32
	for _, diskName := range diskNames {
		d := api.AzureDataDisk{
			Name:               diskName,
			Lun:                pointer.Int32(lun),
			Caching:            "None",
			StorageAccountType: StorageAccountType,
			DiskSizeGB:         20,
		}
		dataDisks = append(dataDisks, d)
		lun++
	}
	b.spec.Properties.StorageProfile.DataDisks = dataDisks
	return b
}

func (b *ProviderSpecBuilder) WithDefaultOsProfile() *ProviderSpecBuilder {
	b.spec.Properties.OsProfile = api.AzureOSProfile{
		AdminUsername:      "core",
		LinuxConfiguration: api.AzureLinuxConfiguration{},
	}
	return b
}

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

func (b *ProviderSpecBuilder) WithTags(tags map[string]string) *ProviderSpecBuilder {
	b.spec.Tags = tags
	return b
}

func (b *ProviderSpecBuilder) Marshal() ([]byte, error) {
	return json.Marshal(b.spec)
}

func (b *ProviderSpecBuilder) Build() api.AzureProviderSpec {
	return b.spec
}
