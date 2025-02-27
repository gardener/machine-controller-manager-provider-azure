// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package access

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// TokenCredentialProvider is a function which gets azcore.TokenCredential using the passed ConnectConfig.
// This allows unit tests to pass their own fake provider for token credentials.
type TokenCredentialProvider func(connectConfig ConnectConfig) (azcore.TokenCredential, error)

// defaultFactory implements Factory interface.
type defaultFactory struct {
	tokenCredentialProvider TokenCredentialProvider
}

// NewDefaultAccessFactory creates a new instance of Factory.
func NewDefaultAccessFactory() Factory {
	return defaultFactory{
		tokenCredentialProvider: GetDefaultTokenCredentials,
	}
}

// GetDefaultTokenCredentials provides the azure token credentials using the ConnectConfig passed as an argument.
func GetDefaultTokenCredentials(connectConfig ConnectConfig) (azcore.TokenCredential, error) {
	if len(connectConfig.WorkloadIdentityTokenFile) > 0 {
		return azidentity.NewWorkloadIdentityCredential(
			&azidentity.WorkloadIdentityCredentialOptions{
				TenantID:      connectConfig.TenantID,
				ClientID:      connectConfig.ClientID,
				TokenFilePath: connectConfig.WorkloadIdentityTokenFile,
				ClientOptions: connectConfig.ClientOptions,
			},
		)
	}

	return azidentity.NewClientSecretCredential(
		connectConfig.TenantID,
		connectConfig.ClientID,
		connectConfig.ClientSecret,
		&azidentity.ClientSecretCredentialOptions{ClientOptions: connectConfig.ClientOptions},
	)
}

func (f defaultFactory) GetResourceGroupsAccess(connectConfig ConnectConfig) (*armresources.ResourceGroupsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armresources.NewResourceGroupsClient(connectConfig.SubscriptionID, tokenCredential, &arm.ClientOptions{ClientOptions: connectConfig.ClientOptions})
}

func (f defaultFactory) GetVirtualMachinesAccess(connectConfig ConnectConfig) (*armcompute.VirtualMachinesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewVirtualMachinesClient(connectConfig.SubscriptionID, tokenCredential, &arm.ClientOptions{ClientOptions: connectConfig.ClientOptions})
}

func (f defaultFactory) GetNetworkInterfacesAccess(connectConfig ConnectConfig) (*armnetwork.InterfacesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armnetwork.NewInterfacesClient(connectConfig.SubscriptionID, tokenCredential, &arm.ClientOptions{ClientOptions: connectConfig.ClientOptions})
}

func (f defaultFactory) GetSubnetAccess(connectConfig ConnectConfig) (*armnetwork.SubnetsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armnetwork.NewSubnetsClient(connectConfig.SubscriptionID, tokenCredential, &arm.ClientOptions{ClientOptions: connectConfig.ClientOptions})
}

func (f defaultFactory) GetDisksAccess(connectConfig ConnectConfig) (*armcompute.DisksClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewDisksClient(connectConfig.SubscriptionID, tokenCredential, &arm.ClientOptions{ClientOptions: connectConfig.ClientOptions})
}

func (f defaultFactory) GetResourceGraphAccess(connectConfig ConnectConfig) (*armresourcegraph.Client, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armresourcegraph.NewClient(tokenCredential, &arm.ClientOptions{ClientOptions: connectConfig.ClientOptions})
}

func (f defaultFactory) GetVirtualMachineImagesAccess(connectConfig ConnectConfig) (*armcompute.VirtualMachineImagesClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armcompute.NewVirtualMachineImagesClient(connectConfig.SubscriptionID, tokenCredential, &arm.ClientOptions{ClientOptions: connectConfig.ClientOptions})
}

func (f defaultFactory) GetMarketPlaceAgreementsAccess(connectConfig ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error) {
	tokenCredential, err := f.tokenCredentialProvider(connectConfig)
	if err != nil {
		return nil, err
	}
	return armmarketplaceordering.NewMarketplaceAgreementsClient(connectConfig.SubscriptionID, tokenCredential, &arm.ClientOptions{ClientOptions: connectConfig.ClientOptions})
}
