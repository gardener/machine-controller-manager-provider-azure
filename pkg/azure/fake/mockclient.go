/*
 * Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute/computeapi"

	"github.com/Azure/azure-sdk-for-go/services/marketplaceordering/mgmt/2015-06-01/marketplaceordering/marketplaceorderingapi"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-04-01/network/networkapi"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis/validation"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
)

// VMClient . . .
type VMClient struct {
	VM computeapi.VirtualMachinesClientAPI
}

// SubnetsClient ...
type SubnetsClient struct {
	Subnet networkapi.SubnetsClientAPI
}

// InterfacesClient ...
type InterfacesClient struct {
	Nic networkapi.InterfacesClientAPI
	ID  string
}

// DisksClient ...
type DisksClient struct {
	Disk computeapi.DisksClientAPI
}

// VirtualMachineImagesClient ...
type VirtualMachineImagesClient struct {
	Images computeapi.VirtualMachineImagesClientAPI
}

// MarketplaceAgreementsClient ...
type MarketplaceAgreementsClient struct {
	Marketplace marketplaceorderingapi.MarketplaceAgreementsClientAPI
}

// FakeAzureDriverClients . . .
type FakeAzureDriverClients struct {
	Subnet      SubnetsClient
	Nic         InterfacesClient
	VM          VMClient
	Disk        DisksClient
	Deployments resources.DeploymentsClient
	Images      VirtualMachineImagesClient
	Marketplace MarketplaceAgreementsClient
}

// SessionProviderInterface ...
type SessionProviderInterface interface {
	Setup(cloudConfig *corev1.Secret) (*FakeAzureDriverClients, error)
}

//PluginSPIImpl is the mock implementation of PluginSPIImpl
type PluginSPIImpl struct {
	SPI               SessionProviderInterface
	AzureProviderSpec *api.AzureProviderSpec
	Secret            *corev1.Secret
}

// NewFakeAzureDriver returns an empty AzureDriver object
func NewFakeAzureDriver(spi SessionProviderInterface) *PluginSPIImpl {
	return &PluginSPIImpl{
		SPI: spi,
	}
}

// CreateOrUpdate ...
func (client VMClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, VMName string, parameters compute.VirtualMachine) (*compute.VirtualMachine, error) {
	var result = &compute.VirtualMachine{}
	result.Name = &VMName
	result.Location = parameters.Location
	return result, nil
}

//Setup creates a compute service instance using the mock
func (ms *PluginSPIImpl) Setup(secret *corev1.Secret) (*FakeAzureDriverClients, error) {

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
func NewClients(subscriptionID, tenantID, clientID, clientSecret string, env azure.Environment) (*FakeAzureDriverClients, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	spToken, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, env.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	authorizer := autorest.NewBearerAuthorizer(spToken)

	subnetClient := SubnetsClient{}

	interfacesClient := InterfacesClient{}

	vmClient := VMClient{}

	vmImagesClient := VirtualMachineImagesClient{}

	diskClient := DisksClient{}

	deploymentsClient := resources.NewDeploymentsClient(subscriptionID)
	deploymentsClient.Authorizer = authorizer

	marketplaceClient := MarketplaceAgreementsClient{}

	return &FakeAzureDriverClients{Subnet: subnetClient, Nic: interfacesClient, VM: vmClient, Disk: diskClient, Deployments: deploymentsClient, Images: vmImagesClient, Marketplace: marketplaceClient}, nil
}

// decodeProviderSpecAndSecret unmarshals the raw providerspec into api.AzureProviderSpec structure
func decodeProviderSpecAndSecret(machineClass *v1alpha1.MachineClass, secret *corev1.Secret) (*api.AzureProviderSpec, error) {
	var providerSpec *api.AzureProviderSpec

	// Extract providerSpec
	err := json.Unmarshal(machineClass.ProviderSpec.Raw, &providerSpec)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	//Validate the Spec and Secrets
	ValidationErr := validation.ValidateAzureSpecNSecret(providerSpec, secret)
	if ValidationErr != nil {
		err = fmt.Errorf("Error while validating ProviderSpec %v", ValidationErr)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return providerSpec, nil
}

// DeleteMachine ...
func (ms *PluginSPIImpl) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {

	providerSpec, err := decodeProviderSpecAndSecret(req.MachineClass, req.Secret)
	ms.AzureProviderSpec = providerSpec

	var (
		vmName            = strings.ToLower(req.Machine.Name)
		resourceGroupName = providerSpec.ResourceGroup
		nicName           = dependencyNameFromVMName(vmName, nicSuffix)
		diskName          = dependencyNameFromVMName(vmName, diskSuffix)
		dataDiskNames     []string
	)
	if providerSpec.Properties.StorageProfile.DataDisks != nil && len(providerSpec.Properties.StorageProfile.DataDisks) > 0 {
		dataDiskNames = getAzureDataDiskNames(providerSpec.Properties.StorageProfile.DataDisks, vmName, dataDiskSuffix)
	}

	clients, err := ms.SPI.Setup(req.Secret)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	err = clients.DeleteVMNicDisks(ctx, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	return &driver.DeleteMachineResponse{}, nil
}
