// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package access

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// ConnectConfig is the configuration required to connect to azure provider.
type ConnectConfig struct {
	// SubscriptionID is a unique ID identifying a subscription.
	SubscriptionID string
	// TenantID is a unique identifier for an active directory tenant.
	TenantID string
	// ClientID is a unique identity assigned by azure active directory to an application.
	ClientID string
	// ClientSecret is a certificate issues for the ClientID.
	// This field is mutually exclusive with WorkloadIdentityTokenFile.
	ClientSecret string
	// WorkloadIdentityTokenFile is the file that a token that is used to be exchanged for Azure credentials.
	// This field is mutually exclusive with ClientSecret.
	WorkloadIdentityTokenFile string
	// ClientOptions are the options to use when connecting with clients.
	ClientOptions policy.ClientOptions
}

// Factory is an access factory providing methods to get facade/access for different resources.
// Azure SDK provides clients for resources, these clients are actually just facades which internally uses another client.
type Factory interface {
	// GetResourceGroupsAccess creates and returns a new instance of armresources.ResourceGroupsClient.
	GetResourceGroupsAccess(connectConfig ConnectConfig) (*armresources.ResourceGroupsClient, error)
	// GetVirtualMachinesAccess creates and returns a new instance of armcompute.VirtualMachinesClient.
	GetVirtualMachinesAccess(connectConfig ConnectConfig) (*armcompute.VirtualMachinesClient, error)
	// GetNetworkInterfacesAccess creates and returns a new instance of armnetwork.InterfacesClient.
	GetNetworkInterfacesAccess(connectConfig ConnectConfig) (*armnetwork.InterfacesClient, error)
	// GetSubnetAccess creates and returns a new instance of armnetwork.SubnetsClient.
	GetSubnetAccess(connectConfig ConnectConfig) (*armnetwork.SubnetsClient, error)
	// GetDisksAccess creates and returns a new instance of armcompute.DisksClient.
	GetDisksAccess(connectConfig ConnectConfig) (*armcompute.DisksClient, error)
	// GetResourceGraphAccess creates and returns a new instance of armresourcegraph.Client.
	GetResourceGraphAccess(connectConfig ConnectConfig) (*armresourcegraph.Client, error)
	// GetVirtualMachineImagesAccess creates and returns a new instance of armcompute.VirtualMachineImagesClient.
	GetVirtualMachineImagesAccess(connectConfig ConnectConfig) (*armcompute.VirtualMachineImagesClient, error)
	// GetMarketPlaceAgreementsAccess creates and returns a new instance of armmarketplaceordering.MarketplaceAgreementsClient.
	GetMarketPlaceAgreementsAccess(connectConfig ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error)
}
