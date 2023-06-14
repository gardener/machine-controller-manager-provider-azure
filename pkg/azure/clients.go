package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/types"
)

// clientFactory implements ClientProvider interface.
type clientFactory struct {
}

// NewClientsProvider creates a new instance of ClientProvider.
func NewClientsProvider() types.ClientProvider {
	return clientFactory{}
}

func (c clientFactory) CreateVirtualMachinesClient(connectConfig types.ConnectConfig) (*armcompute.VirtualMachinesClient, error) {
	tokenCredential, err := connectConfig.CreateTokenCredential()
	if err != nil {
		return nil, err
	}
	factory, err := armcompute.NewClientFactory(connectConfig.SubscriptionID, tokenCredential, nil)
	if err != nil {
		return nil, err
	}
	return factory.NewVirtualMachinesClient(), nil
}

func (c clientFactory) CreateNetworkInterfacesClient(connectConfig types.ConnectConfig) (*armnetwork.InterfacesClient, error) {
	tokenCredential, err := connectConfig.CreateTokenCredential()
	if err != nil {
		return nil, err
	}
	factory, err := armnetwork.NewClientFactory(connectConfig.SubscriptionID, tokenCredential, nil)
	if err != nil {
		return nil, err
	}
	return factory.NewInterfacesClient(), nil
}

func (c clientFactory) CreateResourceGraphClient(connectConfig types.ConnectConfig) (*armresourcegraph.Client, error) {
	tokenCredential, err := connectConfig.CreateTokenCredential()
	if err != nil {
		return nil, err
	}
	factory, err := armresourcegraph.NewClientFactory(tokenCredential, nil)
	if err != nil {
		return nil, err
	}
	return factory.NewClient(), nil
}

func (c clientFactory) CreateImagesClient(connectConfig types.ConnectConfig) (*armcompute.ImagesClient, error) {
	tokenCredential, err := connectConfig.CreateTokenCredential()
	if err != nil {
		return nil, err
	}
	factory, err := armcompute.NewClientFactory(connectConfig.SubscriptionID, tokenCredential, nil)
	if err != nil {
		return nil, err
	}
	return factory.NewImagesClient(), nil
}

func (c clientFactory) CreateMarketPlaceAgreementsClient(connectConfig types.ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error) {
	tokenCredential, err := connectConfig.CreateTokenCredential()
	if err != nil {
		return nil, err
	}
	factory, err := armmarketplaceordering.NewClientFactory(connectConfig.SubscriptionID, tokenCredential, nil)
	if err != nil {
		return nil, err
	}
	return factory.NewMarketplaceAgreementsClient(), nil
}
