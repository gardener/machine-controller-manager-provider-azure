/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package mock has the mock framework of Azure SDK for Go for unit testing
package mock

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/spi"

	computeapi "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute/computeapi"
	marketplaceorderingapi "github.com/Azure/azure-sdk-for-go/services/marketplaceordering/mgmt/2015-06-01/marketplaceordering/marketplaceorderingapi"
	networkapi "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-07-01/network/networkapi"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/mock/mock_computeapi"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/mock/mock_marketplaceorderingapi"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/mock/mock_networkapi"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/mock/mock_resourcesapi"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/spi/resourcesapi"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	gomock "github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
)

// AzureDriverClients . . .
type AzureDriverClients struct {
	Subnet      *mock_networkapi.MockSubnetsClientAPI
	NIC         *mock_networkapi.MockInterfacesClientAPI
	VM          *mock_computeapi.MockVirtualMachinesClientAPI
	Disk        *mock_computeapi.MockDisksClientAPI
	Group       *mock_resourcesapi.MockGroupsClientAPI
	Images      *mock_computeapi.MockVirtualMachineImagesClientAPI
	Marketplace *mock_marketplaceorderingapi.MockMarketplaceAgreementsClientAPI

	// deployments resources.DeploymentsClient
}

// GetVM method is the getter for the Virtual Machines Client from the AzureDriverClients
func (clients *AzureDriverClients) GetVM() computeapi.VirtualMachinesClientAPI {
	return clients.VM
}

// GetVMImpl returns the actual struct implementing the networkapi.InterfacesClientAPI
func (clients *AzureDriverClients) GetVMImpl() compute.VirtualMachinesClient {
	return compute.VirtualMachinesClient{}
}

// GetDisk method is the getter for the Disks Client from the AzureDriverClients
func (clients *AzureDriverClients) GetDisk() computeapi.DisksClientAPI {
	return clients.Disk
}

// GetImages is the getter for the Virtual Machines Images Client from the AzureDriverClients
func (clients *AzureDriverClients) GetImages() computeapi.VirtualMachineImagesClientAPI {
	return clients.Images
}

// GetNic is the getter for the  Network Interfaces Client from the AzureDriverClients
func (clients *AzureDriverClients) GetNic() networkapi.InterfacesClientAPI {
	return clients.NIC
}

// GetNicImpl is the getter for the  Network Interfaces Client from the AzureDriverClients
func (clients *AzureDriverClients) GetNicImpl() network.InterfacesClient {
	return network.InterfacesClient{}
}

// GetSubnet is the getter for the Network Subnets Client from the AzureDriverClients
func (clients *AzureDriverClients) GetSubnet() networkapi.SubnetsClientAPI {
	return clients.Subnet
}

// GetGroup is the getter for the resources Group Client from the AzureDriverClients
func (clients *AzureDriverClients) GetGroup() resourcesapi.GroupsClientAPI {
	return clients.Group
}

// GetMarketplace is the getter for the marketplace agreement client from the AzureDriverClients
func (clients *AzureDriverClients) GetMarketplace() marketplaceorderingapi.MarketplaceAgreementsClientAPI {
	return clients.Marketplace
}

// GetClient is the getter for the autorest Client from the AzureDriverClients
func (clients *AzureDriverClients) GetClient() autorest.Client {
	return autorest.Client{}
}

// GetDeployments is the getter for the resources deployment from the AzureDriverClients
// func (clients *azureDriverClients) GetDeployments() resources.DeploymentsClient {
// 	return clients.deployments
// }

//PluginSPIImpl is the mock implementation of PluginSPIImpl
type PluginSPIImpl struct {
	AzureProviderSpec  *api.AzureProviderSpec
	Secret             *corev1.Secret
	Controller         *gomock.Controller
	azureDriverClients *AzureDriverClients
}

// NewMockPluginSPIImpl ...
func NewMockPluginSPIImpl(controller *gomock.Controller) spi.SessionProviderInterface {
	return &PluginSPIImpl{Controller: controller}
}

//Setup creates a compute service instance using the mock
func (ms *PluginSPIImpl) Setup(secret *corev1.Secret) (spi.AzureDriverClientsInterface, error) {

	if ms.azureDriverClients != nil {
		return ms.azureDriverClients, nil
	}

	var (
		subscriptionID = strings.TrimSpace(string(secret.Data[v1alpha1.AzureSubscriptionID]))
		tenantID       = strings.TrimSpace(string(secret.Data[v1alpha1.AzureTenantID]))
		clientID       = strings.TrimSpace(string(secret.Data[v1alpha1.AzureClientID]))
		clientSecret   = strings.TrimSpace(string(secret.Data[v1alpha1.AzureClientSecret]))
		env            = azure.PublicCloud
	)

	newAzureClients, err := ms.newClients(subscriptionID, tenantID, clientID, clientSecret, env)
	if err != nil {
		return nil, err
	}

	ms.azureDriverClients = newAzureClients
	return newAzureClients, nil
}

// NewClients returns the authenticated Azure clients
func (ms *PluginSPIImpl) newClients(subscriptionID, tenantID, clientID, clientSecret string, env azure.Environment) (*AzureDriverClients, error) {

	subnetClient := mock_networkapi.NewMockSubnetsClientAPI(ms.Controller)
	interfacesClient := mock_networkapi.NewMockInterfacesClientAPI(ms.Controller)
	vmClient := mock_computeapi.NewMockVirtualMachinesClientAPI(ms.Controller)
	vmImagesClient := mock_computeapi.NewMockVirtualMachineImagesClientAPI(ms.Controller)
	diskClient := mock_computeapi.NewMockDisksClientAPI(ms.Controller)
	groupsClients := mock_resourcesapi.NewMockGroupsClientAPI(ms.Controller)
	marketplaceClient := mock_marketplaceorderingapi.NewMockMarketplaceAgreementsClientAPI(ms.Controller)

	// deploymentsClient := resources.NewDeploymentsClient(subscriptionID) // check this subscriptionid

	return &AzureDriverClients{Subnet: subnetClient, NIC: interfacesClient, VM: vmClient, Disk: diskClient, Group: groupsClients, Images: vmImagesClient, Marketplace: marketplaceClient}, nil
}
