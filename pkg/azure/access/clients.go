package access

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// TokenCredentialProvider is a function which gets azcore.TokenCredential using the passed ConnectConfig.
// This allows unit tests to pass their own fake provider for token credentials.
type TokenCredentialProvider func(connectConfig ConnectConfig) (azcore.TokenCredential, error)

// defaultFactory implements Factory interface.
type defaultFactory struct {
	clientOptions           *arm.ClientOptions
	tokenCredentialProvider TokenCredentialProvider
}

// NewDefaultClientsProvider creates a new instance of Factory.
func NewDefaultClientsProvider() Factory {
	return NewClientsProviderWithOptions(nil, GetDefaultTokenCredentials)
}

// NewClientsProviderWithOptions creates an instance of Factory which uses a fixed tokenCredential
// passed as an argument and also takes custom arm.ClientOptions.
func NewClientsProviderWithOptions(clientOptions *arm.ClientOptions, tokenCredentialProvider TokenCredentialProvider) Factory {
	return defaultFactory{
		clientOptions:           clientOptions,
		tokenCredentialProvider: tokenCredentialProvider,
	}
}

func GetDefaultTokenCredentials(connectConfig ConnectConfig) (azcore.TokenCredential, error) {
	return azidentity.NewClientSecretCredential(connectConfig.TenantID, connectConfig.ClientID, connectConfig.ClientSecret, nil)
}

func (f defaultFactory) GetResourceGroupsAccess(connectConfig ConnectConfig) (*armresources.ResourceGroupsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armresources.NewResourceGroupsClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

func (f defaultFactory) GetVirtualMachinesAccess(connectConfig ConnectConfig) (*armcompute.VirtualMachinesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewVirtualMachinesClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

func (f defaultFactory) GetNetworkInterfacesAccess(connectConfig ConnectConfig) (*armnetwork.InterfacesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armnetwork.NewInterfacesClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

func (f defaultFactory) GetSubnetAccess(connectConfig ConnectConfig) (*armnetwork.SubnetsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armnetwork.NewSubnetsClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

func (f defaultFactory) GetDisksAccess(connectConfig ConnectConfig) (*armcompute.DisksClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewDisksClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

func (f defaultFactory) GetResourceGraphAccess(connectConfig ConnectConfig) (*armresourcegraph.Client, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armresourcegraph.NewClient(tokenCredential, f.clientOptions)
}

func (f defaultFactory) GetImagesAccess(connectConfig ConnectConfig) (*armcompute.ImagesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewImagesClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

func (f defaultFactory) GetMarketPlaceAgreementsAccess(connectConfig ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armmarketplaceordering.NewMarketplaceAgreementsClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}
