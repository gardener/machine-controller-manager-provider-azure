/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package spi

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/marketplaceordering/mgmt/marketplaceordering"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	corev1 "k8s.io/api/core/v1"

	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
)

// PluginSPIImpl is the real implementation of SPI interface that makes the calls to the Azure SDK.
type PluginSPIImpl struct{}

// Setup starts a new Azure session
func (ms *PluginSPIImpl) Setup(secret *corev1.Secret) (AzureDriverClientsInterface, error) {
	var (
		subscriptionID = extractCredentialsFromData(secret.Data, api.AzureSubscriptionID, api.AzureAlternativeSubscriptionID)
		tenantID       = extractCredentialsFromData(secret.Data, api.AzureTenantID, api.AzureAlternativeTenantID)
		clientID       = extractCredentialsFromData(secret.Data, api.AzureClientID, api.AzureAlternativeClientID)
		clientSecret   = extractCredentialsFromData(secret.Data, api.AzureClientSecret, api.AzureAlternativeClientSecret)

		env = azure.PublicCloud
	)
	return newClients(subscriptionID, tenantID, clientID, clientSecret, env)
}

// newClients returns the authenticated Azure clients
func newClients(subscriptionID, tenantID, clientID, clientSecret string, env azure.Environment) (*azureDriverClients, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	spToken, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, env.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	authorizer := autorest.NewBearerAuthorizer(spToken)

	subnetClient := network.NewSubnetsClient(subscriptionID)
	subnetClient.Authorizer = authorizer

	interfacesClient := network.NewInterfacesClient(subscriptionID)
	interfacesClient.Authorizer = authorizer

	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = authorizer

	vmImagesClient := compute.NewVirtualMachineImagesClient(subscriptionID)
	vmImagesClient.Authorizer = authorizer

	diskClient := compute.NewDisksClient(subscriptionID)
	diskClient.Authorizer = authorizer

	// deploymentsClient := resources.NewDeploymentsClient(subscriptionID)
	// deploymentsClient.Authorizer = authorizer

	groupClient := resources.NewGroupsClient(subscriptionID)
	groupClient.Authorizer = authorizer

	marketplaceClient := marketplaceordering.NewMarketplaceAgreementsClient(subscriptionID)
	marketplaceClient.Authorizer = authorizer

	return &azureDriverClients{subnet: subnetClient, nic: interfacesClient, vm: vmClient, disk: diskClient, group: groupClient, images: vmImagesClient, marketplace: marketplaceClient}, nil

	// return &azureDriverClients{subnet: subnetClient, nic: interfacesClient, vm: vmClient, disk: diskClient, deployments: deploymentsClient, group: groupClient, images: vmImagesClient, marketplace: marketplaceClient}, nil
}

// extractCredentialsFromData extracts and trims a value from the given data map. The first key that exists is being
// returned, otherwise, the next key is tried, etc. If no key exists then an empty string is returned.
func extractCredentialsFromData(data map[string][]byte, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			return strings.TrimSpace(string(val))
		}
	}
	return ""
}
