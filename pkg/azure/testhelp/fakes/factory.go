package fakes

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	fakenetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	fakeresourcegraph "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	fakearmresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
)

func NewFactory(resourceGroup string) *Factory {
	return &Factory{
		resourceGroup: resourceGroup,
	}
}

type Factory struct {
	resourceGroup               string
	VMAccess                    *armcompute.VirtualMachinesClient
	ResourceGroupAccess         *armresources.ResourceGroupsClient
	InterfaceAccess             *armnetwork.InterfacesClient
	SubnetAccess                *armnetwork.SubnetsClient
	DisksAccess                 *armcompute.DisksClient
	ResourceGraphAccess         *armresourcegraph.Client
	VMImageAccess               *armcompute.VirtualMachineImagesClient
	MarketplaceAgreementsAccess *armmarketplaceordering.MarketplaceAgreementsClient
}

// Fake implementation methods of access.Factory interface.
// --------------------------------------------------------------------------------------------

func (f *Factory) GetVirtualMachinesAccess(_ access.ConnectConfig) (*armcompute.VirtualMachinesClient, error) {
	return f.VMAccess, nil
}
func (f *Factory) GetResourceGroupsAccess(_ access.ConnectConfig) (*armresources.ResourceGroupsClient, error) {
	return f.ResourceGroupAccess, nil
}
func (f *Factory) GetNetworkInterfacesAccess(_ access.ConnectConfig) (*armnetwork.InterfacesClient, error) {
	return f.InterfaceAccess, nil
}
func (f *Factory) GetSubnetAccess(_ access.ConnectConfig) (*armnetwork.SubnetsClient, error) {
	return f.SubnetAccess, nil
}
func (f *Factory) GetDisksAccess(_ access.ConnectConfig) (*armcompute.DisksClient, error) {
	return f.DisksAccess, nil
}
func (f *Factory) GetResourceGraphAccess(_ access.ConnectConfig) (*armresourcegraph.Client, error) {
	return f.ResourceGraphAccess, nil
}
func (f *Factory) GetVirtualMachineImagesAccess(_ access.ConnectConfig) (*armcompute.VirtualMachineImagesClient, error) {
	return f.VMImageAccess, nil
}
func (f *Factory) GetMarketPlaceAgreementsAccess(_ access.ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error) {
	return f.MarketplaceAgreementsAccess, nil
}

// --------------------------------------------------------------------------------------------
// Builder methods to allow partial initialization of fake Factory.
// --------------------------------------------------------------------------------------------

func (f *Factory) NewVirtualMachineAccessBuilder() *VMAccessBuilder {
	return &VMAccessBuilder{
		vmServer: fakecompute.VirtualMachinesServer{},
	}
}

func (f *Factory) NewResourceGroupsAccessBuilder() *ResourceGroupsAccessBuilder {
	return &ResourceGroupsAccessBuilder{
		rg:       f.resourceGroup,
		rgServer: fakearmresources.ResourceGroupsServer{},
	}
}

func (f *Factory) NewNICAccessBuilder() *NICAccessBuilder {
	return &NICAccessBuilder{
		nicServer: fakenetwork.InterfacesServer{},
	}
}

func (f *Factory) NewDiskAccessBuilder() *DiskAccessBuilder {
	return &DiskAccessBuilder{
		diskServer: fakecompute.DisksServer{},
	}
}

func (f *Factory) NewResourceGraphAccessBuilder() *ResourceGraphAccessBuilder {
	return &ResourceGraphAccessBuilder{
		resourceGraphServer: fakeresourcegraph.Server{},
	}
}

func (f *Factory) WithVirtualMachineAccess(vmAccess *armcompute.VirtualMachinesClient) *Factory {
	f.VMAccess = vmAccess
	return f
}
func (f *Factory) WithResourceGroupsAccess(rgAccess *armresources.ResourceGroupsClient) *Factory {
	f.ResourceGroupAccess = rgAccess
	return f
}
func (f *Factory) WithNetworkInterfacesAccess(nwiAccess *armnetwork.InterfacesClient) *Factory {
	f.InterfaceAccess = nwiAccess
	return f
}
func (f *Factory) WithSubnetAccess(subnetAccess *armnetwork.SubnetsClient) *Factory {
	f.SubnetAccess = subnetAccess
	return f
}
func (f *Factory) WithDisksAccess(diskClient *armcompute.DisksClient) *Factory {
	f.DisksAccess = diskClient
	return f
}
func (f *Factory) WithResourceGraphAccess(rgAccess *armresourcegraph.Client) *Factory {
	f.ResourceGraphAccess = rgAccess
	return f
}
func (f *Factory) WithVirtualMachineImagesAccess(vmImageAccess *armcompute.VirtualMachineImagesClient) *Factory {
	f.VMImageAccess = vmImageAccess
	return f
}
func (f *Factory) WithMarketPlaceAgreementsAccess(mpaAccess *armmarketplaceordering.MarketplaceAgreementsClient) *Factory {
	f.MarketplaceAgreementsAccess = mpaAccess
	return f
}
