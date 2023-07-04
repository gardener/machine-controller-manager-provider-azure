package test

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
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
			Location:      "westeurope",
			ResourceGroup: resourceGroup,
			Properties: api.AzureVirtualMachineProperties{
				Zone: to.Ptr(1),
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

func (b *ProviderSpecBuilder) WithDefaultHardwareProfile() *ProviderSpecBuilder {
	b.spec.Properties.HardwareProfile = api.AzureHardwareProfile{VMSize: "Standard_DS2_v2"}
	return b
}

func (b *ProviderSpecBuilder) WithDefaultStorageProfile() *ProviderSpecBuilder {
	b.spec.Properties.StorageProfile = api.AzureStorageProfile{
		ImageReference: api.AzureImageReference{
			URN: to.Ptr("sap:gardenlinux:greatest:184.0.0"),
		},
		OsDisk: api.AzureOSDisk{
			Caching: "None",
			ManagedDisk: api.AzureManagedDiskParameters{
				StorageAccountType: "StandardSSD_LRS",
			},
			DiskSizeGB:   50,
			CreateOption: "FromImage",
		},
	}
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
