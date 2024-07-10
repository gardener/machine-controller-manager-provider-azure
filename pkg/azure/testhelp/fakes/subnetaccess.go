// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	fakenetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

// SubnetAccessBuilder is a builder for Subnet access.
type SubnetAccessBuilder struct {
	clusterState    *ClusterState
	server          fakenetwork.SubnetsServer
	apiBehaviorSpec *APIBehaviorSpec
}

// WithClusterState initializes builder with a ClusterState.
func (b *SubnetAccessBuilder) WithClusterState(clusterState *ClusterState) *SubnetAccessBuilder {
	b.clusterState = clusterState
	return b
}

// WithAPIBehaviorSpec initializes the builder with a APIBehaviorSpec.
func (b *SubnetAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *SubnetAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

// withGet implements the Get method of armnetwork.SubnetsClient and initializes the backing fake server's Get method with the anonymous function implementation.
func (b *SubnetAccessBuilder) withGet() *SubnetAccessBuilder {
	b.server.Get = func(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, _ *armnetwork.SubnetsClientGetOptions) (resp azfake.Responder[armnetwork.SubnetsClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.ProviderSpec.ResourceGroup, to.Ptr(utils.SubnetResourceType), testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if !b.clusterState.ResourceGroupExists(resourceGroupName) {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}
		subnet := b.clusterState.GetSubnet(resourceGroupName, subnetName, virtualNetworkName)
		if subnet == nil {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeSubnetNotFound))
			return
		}
		resp.SetResponse(http.StatusOK, armnetwork.SubnetsClientGetResponse{Subnet: *subnet}, nil)
		return
	}
	return b
}

// Build builds armnetwork.SubnetsClient.
func (b *SubnetAccessBuilder) Build() (*armnetwork.SubnetsClient, error) {
	b.withGet()
	return armnetwork.NewSubnetsClient(testhelp.SubscriptionID, &azfake.TokenCredential{}, &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakenetwork.NewSubnetsServerTransport(&b.server),
		},
	})
}
