package access

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// ConnectConfig is the configuration required for a access to connect to azure.
type ConnectConfig struct {
	// SubscriptionID
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
}

// Factory is a factory providing methods to get clients for different resources.
type Factory interface {
	GetResourceGroupsAccess(connectConfig ConnectConfig) (*armresources.ResourceGroupsClient, error)
	GetVirtualMachinesAccess(connectConfig ConnectConfig) (*armcompute.VirtualMachinesClient, error)
	GetNetworkInterfacesAccess(connectConfig ConnectConfig) (*armnetwork.InterfacesClient, error)
	GetSubnetAccess(connectConfig ConnectConfig) (*armnetwork.SubnetsClient, error)
	GetDisksAccess(connectConfig ConnectConfig) (*armcompute.DisksClient, error)
	GetResourceGraphAccess(connectConfig ConnectConfig) (*armresourcegraph.Client, error)
	GetImagesAccess(connectConfig ConnectConfig) (*armcompute.ImagesClient, error)
	GetMarketPlaceAgreementsAccess(connectConfig ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error)
}
