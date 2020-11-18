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
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/client"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// MachinePlugin implements the driver.Driver
// It also implements the PluginSPI interface
type MachinePlugin struct {
	SPI SessionProviderInterface
}

// PluginSPIImpl is the real implementation of SPI interface that makes the calls to the Azure SDK.
type PluginSPIImpl struct{}

// Setup starts a new Azure session
func (ms *PluginSPIImpl) Setup(secret *corev1.Secret) (*client.AzureDriverClients, error) {
	var (
		subscriptionID = strings.TrimSpace(string(secret.Data[v1alpha1.AzureSubscriptionID]))
		tenantID       = strings.TrimSpace(string(secret.Data[v1alpha1.AzureTenantID]))
		clientID       = strings.TrimSpace(string(secret.Data[v1alpha1.AzureClientID]))
		clientSecret   = strings.TrimSpace(string(secret.Data[v1alpha1.AzureClientSecret]))
		env            = azure.PublicCloud
	)
	newClients, err := NewClients(subscriptionID, tenantID, clientID, clientSecret, env)
	if err != nil {
		return nil, err
	}
	return newClients, nil
}

// NewClients returns the authenticated Azure clients
func NewClients(subscriptionID, tenantID, clientID, clientSecret string, env azure.Environment) (*client.AzureDriverClients, error) {
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

	deploymentsClient := resources.NewDeploymentsClient(subscriptionID)
	deploymentsClient.Authorizer = authorizer

	marketplaceClient := marketplaceordering.NewMarketplaceAgreementsClient(subscriptionID)
	marketplaceClient.Authorizer = authorizer

	return &client.AzureDriverClients{Subnet: subnetClient, Nic: interfacesClient, VM: vmClient, Disk: diskClient, Deployments: deploymentsClient, Images: vmImagesClient, Marketplace: marketplaceClient}, nil
}
