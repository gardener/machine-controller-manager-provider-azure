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
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	fakenetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
)

type SubnetAccessBuilder struct {
	clusterState    *ClusterState
	server          fakenetwork.SubnetsServer
	apiBehaviorSpec *APIBehaviorSpec
}

func (b *SubnetAccessBuilder) WithClusterState(clusterState *ClusterState) *SubnetAccessBuilder {
	b.clusterState = clusterState
	return b
}

func (b *SubnetAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *SubnetAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

func (b *SubnetAccessBuilder) withGet() *SubnetAccessBuilder {
	b.server.Get = func(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, options *armnetwork.SubnetsClientGetOptions) (resp azfake.Responder[armnetwork.SubnetsClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.ProviderSpec.ResourceGroup, to.Ptr(SubnetResourceType), testhelp.AccessMethodGet)
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

func (b *SubnetAccessBuilder) Build() (*armnetwork.SubnetsClient, error) {
	b.withGet()
	return armnetwork.NewSubnetsClient(testhelp.SubscriptionID, azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakenetwork.NewSubnetsServerTransport(&b.server),
		},
	})
}
