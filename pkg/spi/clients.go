/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package spi implements the auxilliary methods for AzureDriverClient
package spi

import (
	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/marketplaceordering/mgmt/marketplaceordering"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	computeapi "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute/computeapi"
	marketplaceorderingapi "github.com/Azure/azure-sdk-for-go/services/marketplaceordering/mgmt/2015-06-01/marketplaceordering/marketplaceorderingapi"
	networkapi "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-07-01/network/networkapi"
	"github.com/Azure/go-autorest/autorest"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/spi/resourcesapi"
)

const (
	prometheusServiceSubnet = "subnet"
	prometheusServiceVM     = "virtual_machine"
	prometheusServiceNIC    = "network_interfaces"
	prometheusServiceDisk   = "disks"
)

// AzureDriverClientsInterface is the interfaces to be implemented
// by the AzureDriverClients to get and refer the respective clients
type AzureDriverClientsInterface interface {

	// GetSubnet() is the getter for the Azure Subnets Client
	GetSubnet() networkapi.SubnetsClientAPI

	// GetNic() is the getter for the Azure Interfaces Client
	GetNic() networkapi.InterfacesClientAPI

	// GetNicImpl returns the actual struct implementing the networkapi.InterfacesClientAPI
	GetNicImpl() network.InterfacesClient

	// GetVM() is the getter for the Azure Virtual Machines Client
	GetVM() computeapi.VirtualMachinesClientAPI

	// GetVMImpl returns the actual struct implementing the computeapi.VirtualMachinesClientAPI
	GetVMImpl() compute.VirtualMachinesClient

	// GetDisk() is the getter for the Azure Disks Client
	GetDisk() computeapi.DisksClientAPI

	// GetImages() is the getter for the Azure Virtual Machines Images Client
	GetImages() computeapi.VirtualMachineImagesClientAPI

	// GetDeployments() is the getter for the Azure Deployment Client
	// GetDeployments() resources.DeploymentsClient

	// GetGroup is the getter for the Azure Groups Client
	GetGroup() resourcesapi.GroupsClientAPI

	// GetMarketplace() is the getter for the Azure Marketplace Agreement Client
	GetMarketplace() marketplaceorderingapi.MarketplaceAgreementsClientAPI

	// GetClient() is the getter of the Azure autorest client
	GetClient() autorest.Client
}

// azureDriverClients . . .
type azureDriverClients struct {
	subnet      network.SubnetsClient
	nic         network.InterfacesClient
	vm          compute.VirtualMachinesClient
	disk        compute.DisksClient
	images      compute.VirtualMachineImagesClient
	group       resources.GroupsClient
	marketplace marketplaceordering.MarketplaceAgreementsClient

	// commenting the below deployments attribute as I do not see an active usage of it in the core
	// deployments resources.DeploymentsClient

}

// GetVM method is the getter for the Virtual Machines Client from the AzureDriverClients
func (clients *azureDriverClients) GetVM() computeapi.VirtualMachinesClientAPI {
	return clients.vm
}

// GetVMImpl returns the actual struct implementing the computeapi.VirtualMachinesClientAPI
func (clients *azureDriverClients) GetVMImpl() compute.VirtualMachinesClient {
	return clients.vm
}

// GetDisk method is the getter for the Disks Client from the AzureDriverClients
func (clients *azureDriverClients) GetDisk() computeapi.DisksClientAPI {
	return clients.disk
}

// GetImages is the getter for the Virtual Machines Images Client from the AzureDriverClients
func (clients *azureDriverClients) GetImages() computeapi.VirtualMachineImagesClientAPI {
	return clients.images
}

// GetNic is the getter for the  Network Interfaces Client from the AzureDriverClients
func (clients *azureDriverClients) GetNic() networkapi.InterfacesClientAPI {
	return clients.nic
}

// GetNicImpl returns the actual struct implementing the networkapi.InterfacesClientAPI
func (clients *azureDriverClients) GetNicImpl() network.InterfacesClient {
	return clients.nic
}

// GetSubnet is the getter for the Network Subnets Client from the AzureDriverClients
func (clients *azureDriverClients) GetSubnet() networkapi.SubnetsClientAPI {
	return clients.subnet
}

// GetDeployments is the getter for the resources deployment from the AzureDriverClients
// func (clients *azureDriverClients) GetDeployments() resources.DeploymentsClient {
// 	return clients.deployments
// }

// GetGroup is the getter for the resources Group Client from the AzureDriverClients
func (clients *azureDriverClients) GetGroup() resourcesapi.GroupsClientAPI {
	return clients.group
}

// GetMarketplace is the getter for the marketplace agreement client from the AzureDriverClients
func (clients *azureDriverClients) GetMarketplace() marketplaceorderingapi.MarketplaceAgreementsClientAPI {
	return clients.marketplace
}

// GetClient is the getter for the autorest Client from the AzureDriverClients
func (clients *azureDriverClients) GetClient() autorest.Client {
	return clients.GetVM().(compute.VirtualMachinesClient).BaseClient.Client
}
