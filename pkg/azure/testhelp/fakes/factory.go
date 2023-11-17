// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fakes

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	fakemktplaceordering "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	fakenetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	fakeresourcegraph "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	fakearmresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
)

// NewFactory creates a new Factory.
func NewFactory(resourceGroup string) *Factory {
	return &Factory{
		resourceGroup: resourceGroup,
	}
}

// Factory captures all resource accesses and a default resource group known and shared by all accesses.
type Factory struct {
	resourceGroup string
	// VMAccess provides access for VirtualMachines.
	VMAccess *armcompute.VirtualMachinesClient
	// ResourceGroupAccess provides access to resource groups.
	ResourceGroupAccess *armresources.ResourceGroupsClient
	// InterfaceAccess provides access to network interfaces.
	InterfaceAccess *armnetwork.InterfacesClient
	// SubnetAccess provides access to subnets.
	SubnetAccess *armnetwork.SubnetsClient
	// DiskAccess provides access to disks.
	DisksAccess *armcompute.DisksClient
	// ResourceGraphAccess provides resource graph querying capabilities using KUSTO.
	ResourceGraphAccess *armresourcegraph.Client
	// VMImageAccess provides access to VM Images.
	VMImageAccess *armcompute.VirtualMachineImagesClient
	// MarketplaceAgreementsAccess provides access to market-place ordering agreements.
	MarketplaceAgreementsAccess *armmarketplaceordering.MarketplaceAgreementsClient
}

// Fake implementation methods of access.Factory interface.
// --------------------------------------------------------------------------------------------

// GetVirtualMachinesAccess gets the configured virtual machine access.
func (f *Factory) GetVirtualMachinesAccess(_ access.ConnectConfig) (*armcompute.VirtualMachinesClient, error) {
	return f.VMAccess, nil
}

// GetResourceGroupsAccess gets the configured resource group access.
func (f *Factory) GetResourceGroupsAccess(_ access.ConnectConfig) (*armresources.ResourceGroupsClient, error) {
	return f.ResourceGroupAccess, nil
}

// GetNetworkInterfacesAccess gets the configured network interface access.
func (f *Factory) GetNetworkInterfacesAccess(_ access.ConnectConfig) (*armnetwork.InterfacesClient, error) {
	return f.InterfaceAccess, nil
}

// GetSubnetAccess gets the configured subnet access.
func (f *Factory) GetSubnetAccess(_ access.ConnectConfig) (*armnetwork.SubnetsClient, error) {
	return f.SubnetAccess, nil
}

// GetDisksAccess gets the configured disk access.
func (f *Factory) GetDisksAccess(_ access.ConnectConfig) (*armcompute.DisksClient, error) {
	return f.DisksAccess, nil
}

// GetResourceGraphAccess gets the configured resource graph access.
func (f *Factory) GetResourceGraphAccess(_ access.ConnectConfig) (*armresourcegraph.Client, error) {
	return f.ResourceGraphAccess, nil
}

// GetVirtualMachineImagesAccess gets the configured access for VM Images.
func (f *Factory) GetVirtualMachineImagesAccess(_ access.ConnectConfig) (*armcompute.VirtualMachineImagesClient, error) {
	return f.VMImageAccess, nil
}

// GetMarketPlaceAgreementsAccess gets the configured access for market-place agreements.
func (f *Factory) GetMarketPlaceAgreementsAccess(_ access.ConnectConfig) (*armmarketplaceordering.MarketplaceAgreementsClient, error) {
	return f.MarketplaceAgreementsAccess, nil
}

// --------------------------------------------------------------------------------------------
// Builder methods to allow partial initialization of fake Factory.
// --------------------------------------------------------------------------------------------

// NewVirtualMachineAccessBuilder creates a new VMAccessBuilder.
func (f *Factory) NewVirtualMachineAccessBuilder() *VMAccessBuilder {
	return &VMAccessBuilder{
		server: fakecompute.VirtualMachinesServer{},
	}
}

// NewResourceGroupsAccessBuilder creates a new ResourceGroupsAccessBuilder initializing the shared resource group.
func (f *Factory) NewResourceGroupsAccessBuilder() *ResourceGroupsAccessBuilder {
	return &ResourceGroupsAccessBuilder{
		rg:     f.resourceGroup,
		server: fakearmresources.ResourceGroupsServer{},
	}
}

// NewNICAccessBuilder creates a new NICAccessBuilder.
func (f *Factory) NewNICAccessBuilder() *NICAccessBuilder {
	return &NICAccessBuilder{
		server: fakenetwork.InterfacesServer{},
	}
}

// NewDiskAccessBuilder creates a new DiskAccessBuilder.
func (f *Factory) NewDiskAccessBuilder() *DiskAccessBuilder {
	return &DiskAccessBuilder{
		server: fakecompute.DisksServer{},
	}
}

// NewResourceGraphAccessBuilder creates a new ResourceGraphAccessBuilder.
func (f *Factory) NewResourceGraphAccessBuilder() *ResourceGraphAccessBuilder {
	return &ResourceGraphAccessBuilder{
		server: fakeresourcegraph.Server{},
	}
}

// NewSubnetAccessBuilder creates a new SubnetAccessBuilder.
func (f *Factory) NewSubnetAccessBuilder() *SubnetAccessBuilder {
	return &SubnetAccessBuilder{
		server: fakenetwork.SubnetsServer{},
	}
}

// NewImageAccessBuilder creates a new ImageAccessBuilder.
func (f *Factory) NewImageAccessBuilder() *ImageAccessBuilder {
	return &ImageAccessBuilder{
		server: fakecompute.VirtualMachineImagesServer{},
	}
}

// NewMarketPlaceAgreementAccessBuilder create a new MarketPlaceAgreementAccessBuilder.
func (f *Factory) NewMarketPlaceAgreementAccessBuilder() *MarketPlaceAgreementAccessBuilder {
	return &MarketPlaceAgreementAccessBuilder{
		server: fakemktplaceordering.MarketplaceAgreementsServer{},
	}
}

// WithVirtualMachineAccess initializes Factory with VM access.
func (f *Factory) WithVirtualMachineAccess(vmAccess *armcompute.VirtualMachinesClient) *Factory {
	f.VMAccess = vmAccess
	return f
}

// WithResourceGroupsAccess initializes Factory with Resource Groups access.
func (f *Factory) WithResourceGroupsAccess(rgAccess *armresources.ResourceGroupsClient) *Factory {
	f.ResourceGroupAccess = rgAccess
	return f
}

// WithNetworkInterfacesAccess initializes Factory with Network Interface access.
func (f *Factory) WithNetworkInterfacesAccess(nwiAccess *armnetwork.InterfacesClient) *Factory {
	f.InterfaceAccess = nwiAccess
	return f
}

// WithSubnetAccess initializes Factory with Subnet access.
func (f *Factory) WithSubnetAccess(subnetAccess *armnetwork.SubnetsClient) *Factory {
	f.SubnetAccess = subnetAccess
	return f
}

// WithDisksAccess initializes Factory with Disk access.
func (f *Factory) WithDisksAccess(diskClient *armcompute.DisksClient) *Factory {
	f.DisksAccess = diskClient
	return f
}

// WithResourceGraphAccess initializes Factory with Resource Graph access.
func (f *Factory) WithResourceGraphAccess(rgAccess *armresourcegraph.Client) *Factory {
	f.ResourceGraphAccess = rgAccess
	return f
}

// WithVirtualMachineImagesAccess initializes Factory with VM Image access.
func (f *Factory) WithVirtualMachineImagesAccess(vmImageAccess *armcompute.VirtualMachineImagesClient) *Factory {
	f.VMImageAccess = vmImageAccess
	return f
}

// WithMarketPlaceAgreementsAccess initializes Factory with MarketPlace Agreements access.
func (f *Factory) WithMarketPlaceAgreementsAccess(mpaAccess *armmarketplaceordering.MarketplaceAgreementsClient) *Factory {
	f.MarketplaceAgreementsAccess = mpaAccess
	return f
}
