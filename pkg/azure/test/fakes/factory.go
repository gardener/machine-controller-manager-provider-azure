package fakes

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
)

type APIBehaviorOptions struct {
	TimeoutAfter *time.Duration
}

func NewFactory(resourceGroup string) *Factory {
	return &Factory{
		resourceGroup: resourceGroup,
	}
}

type Factory struct {
	resourceGroup        string
	vmAccess             *armcompute.VirtualMachinesClient
	resourceGroupsAccess *armresources.ResourceGroupsClient
	nwiAccess            *armnetwork.InterfacesClient
	subnetAccess         *armnetwork.SubnetsClient
	disksClient          *armcompute.DisksClient
	resourceGraphAccess  *armresourcegraph.Client
	imagesAccess         *armcompute.ImagesClient
	mpaAccess            *armmarketplaceordering.MarketplaceAgreementsClient
}

// Fake implementation methods of access.Factory interface.
// --------------------------------------------------------------------------------------------

func (f *Factory) GetVirtualMachinesAccess(_ access.ConnectConfig) (*armcompute.VirtualMachinesClient, error) {
	return f.vmAccess, nil
}
func (f *Factory) GetResourceGroupsAccess(_ access.ConnectConfig) (*armresources.ResourceGroupsClient, error) {
	return f.resourceGroupsAccess, nil
}
func (f *Factory) GetNetworkInterfacesAccess(_ access.ConnectConfig) (*armnetwork.InterfacesClient, error) {
	return f.nwiAccess, nil
}
func (f *Factory) GetSubnetAccess(_ access.ConnectConfig) (*armnetwork.SubnetsClient, error) {
	return f.subnetAccess, nil
}
func (f *Factory) GetDisksAccess(_ access.ConnectConfig) (*armcompute.DisksClient, error) {
	return f.disksClient, nil
}
func (f *Factory) GetResourceGraphAccess(_ access.ConnectConfig) (*armresourcegraph.Client, error) {
	return f.resourceGraphAccess, nil
}
func (f *Factory) GetImagesAccess(_ access.ConnectConfig) (*armcompute.ImagesClient, error) {
	return f.imagesAccess, nil
}
func (f *Factory) GetMarketPlaceAgreementsAccess(_ access.ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error) {
	return f.mpaAccess, nil
}

// --------------------------------------------------------------------------------------------

// Builder methods to allow partial initialization of fake Factory.
// --------------------------------------------------------------------------------------------

func (f *Factory) NewVirtualMachineAccessBuilder() *VMAccessBuilder {
	return &VMAccessBuilder{
		resourceGroup: f.resourceGroup,
		vmServer:      fakecompute.VirtualMachinesServer{},
	}
}

func (f *Factory) NewResourceGroupsAccessBuilder() *ResourceGroupsAccessBuilder {
	return &ResourceGroupsAccessBuilder{rg: f.resourceGroup}
}

func (f *Factory) WithVirtualMachineAccess(vmAccess *armcompute.VirtualMachinesClient) *Factory {
	f.vmAccess = vmAccess
	return f
}
func (f *Factory) WithResourceGroupsAccess(rgAccess *armresources.ResourceGroupsClient) *Factory {
	f.resourceGroupsAccess = rgAccess
	return f
}
func (f *Factory) WithNetworkInterfacesAccess(nwiAccess *armnetwork.InterfacesClient) *Factory {
	f.nwiAccess = nwiAccess
	return f
}
func (f *Factory) WithSubnetAccess(subnetAccess *armnetwork.SubnetsClient) *Factory {
	f.subnetAccess = subnetAccess
	return f
}
func (f *Factory) WithDisksAccess(diskClient *armcompute.DisksClient) *Factory {
	f.disksClient = diskClient
	return f
}
func (f *Factory) WithResourceGraphAccess(rgAccess *armresourcegraph.Client) *Factory {
	f.resourceGraphAccess = rgAccess
	return f
}
func (f *Factory) WithImagesAccess(imagesAccess *armcompute.ImagesClient) *Factory {
	f.imagesAccess = imagesAccess
	return f
}
func (f *Factory) WithMarketPlaceAgreementsAccess(mpaAccess *armmarketplaceordering.MarketplaceAgreementsClient) *Factory {
	f.mpaAccess = mpaAccess
	return f
}

// --------------------------------------------------------------------------------------------
