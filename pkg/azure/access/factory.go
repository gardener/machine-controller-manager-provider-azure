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

// NewDefaultAccessFactory creates a new instance of Factory.
func NewDefaultAccessFactory() Factory {
	return NewAccessFactoryWithOptions(nil, GetDefaultTokenCredentials)
}

// NewAccessFactoryWithOptions creates an instance of Factory which uses a fixed tokenCredential
// passed as an argument and also takes custom arm.ClientOptions.
func NewAccessFactoryWithOptions(clientOptions *arm.ClientOptions, tokenCredentialProvider TokenCredentialProvider) Factory {
	return defaultFactory{
		clientOptions:           clientOptions,
		tokenCredentialProvider: tokenCredentialProvider,
	}
}

// GetDefaultTokenCredentials provides the azure token credentials using the ConnectConfig passed as an argument.
func GetDefaultTokenCredentials(connectConfig ConnectConfig) (azcore.TokenCredential, error) {
	return azidentity.NewClientSecretCredential(connectConfig.TenantID, connectConfig.ClientID, connectConfig.ClientSecret, nil)
}

// GetResourceGroupsAccess creates and returns a new instance of armresources.ResourceGroupsClient.
func (f defaultFactory) GetResourceGroupsAccess(connectConfig ConnectConfig) (*armresources.ResourceGroupsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armresources.NewResourceGroupsClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

// GetVirtualMachinesAccess creates and returns a new instance of armcompute.VirtualMachinesClient.
func (f defaultFactory) GetVirtualMachinesAccess(connectConfig ConnectConfig) (*armcompute.VirtualMachinesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewVirtualMachinesClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

// GetNetworkInterfacesAccess creates and returns a new instance of armnetwork.InterfacesClient.
func (f defaultFactory) GetNetworkInterfacesAccess(connectConfig ConnectConfig) (*armnetwork.InterfacesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armnetwork.NewInterfacesClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

// GetSubnetAccess creates and returns a new instance of armnetwork.SubnetsClient.
func (f defaultFactory) GetSubnetAccess(connectConfig ConnectConfig) (*armnetwork.SubnetsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armnetwork.NewSubnetsClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

// GetDisksAccess creates and returns a new instance of armcompute.DisksClient.
func (f defaultFactory) GetDisksAccess(connectConfig ConnectConfig) (*armcompute.DisksClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewDisksClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

// GetResourceGraphAccess creates and returns a new instance of armresourcegraph.Client.
func (f defaultFactory) GetResourceGraphAccess(connectConfig ConnectConfig) (*armresourcegraph.Client, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armresourcegraph.NewClient(tokenCredential, f.clientOptions)
}

// GetVirtualMachineImagesAccess creates and returns a new instance of armcompute.VirtualMachineImagesClient.
func (f defaultFactory) GetVirtualMachineImagesAccess(connectConfig ConnectConfig) (*armcompute.VirtualMachineImagesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewVirtualMachineImagesClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}

// GetMarketPlaceAgreementsAccess creates and returns a new instance of armmarketplaceordering.MarketplaceAgreementsClient.
func (f defaultFactory) GetMarketPlaceAgreementsAccess(connectConfig ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armmarketplaceordering.NewMarketplaceAgreementsClient(connectConfig.SubscriptionID, tokenCredential, f.clientOptions)
}
