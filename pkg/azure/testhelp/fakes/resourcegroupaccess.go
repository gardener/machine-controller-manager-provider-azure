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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	fakearmresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
)

// ResourceGroupsAccessBuilder is a builder for Resource Groups access.
type ResourceGroupsAccessBuilder struct {
	rg              string
	server          fakearmresources.ResourceGroupsServer
	apiBehaviorSpec *APIBehaviorSpec
}

// WithAPIBehaviorSpec initializes the builder with a APIBehaviorSpec.
func (b *ResourceGroupsAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *ResourceGroupsAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

// withCheckExistence implements the CheckExistence method of armresources.ResourceGroupsClient and initializes the backing fake server's CheckExistence method with the anonymous function implementation.
func (b *ResourceGroupsAccessBuilder) withCheckExistence() *ResourceGroupsAccessBuilder {
	b.server.CheckExistence = func(ctx context.Context, resourceGroupName string, options *armresources.ResourceGroupsClientCheckExistenceOptions) (resp azfake.Responder[armresources.ResourceGroupsClientCheckExistenceResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, resourceGroupName, testhelp.AccessMethodCheckExistence)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.rg != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}
		rgResponse := armresources.ResourceGroupsClientCheckExistenceResponse{Success: true}
		resp.SetResponse(http.StatusNoContent, rgResponse, nil)
		return
	}
	return b
}

// Build builds armresources.ResourceGroupsClient.
func (b *ResourceGroupsAccessBuilder) Build() (*armresources.ResourceGroupsClient, error) {
	b.withCheckExistence()
	return armresources.NewResourceGroupsClient(
		testhelp.SubscriptionID,
		azfake.NewTokenCredential(),
		&arm.ClientOptions{
			ClientOptions: policy.ClientOptions{
				Transport: fakearmresources.NewResourceGroupsServerTransport(&b.server),
			},
		},
	)
}
