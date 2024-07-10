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
	b.server.CheckExistence = func(ctx context.Context, resourceGroupName string, _ *armresources.ResourceGroupsClientCheckExistenceOptions) (resp azfake.Responder[armresources.ResourceGroupsClientCheckExistenceResponse], errResp azfake.ErrorResponder) {
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
		&azfake.TokenCredential{},
		&arm.ClientOptions{
			ClientOptions: policy.ClientOptions{
				Transport: fakearmresources.NewResourceGroupsServerTransport(&b.server),
			},
		},
	)
}
