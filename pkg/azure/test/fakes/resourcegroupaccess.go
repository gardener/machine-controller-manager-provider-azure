package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	fakearmresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test"
)

type ResourceGroupsAccessBuilder struct {
	rg              string
	rgServer        fakearmresources.ResourceGroupsServer
	apiBehaviorSpec *APIBehaviorSpec
}

func (b *ResourceGroupsAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *ResourceGroupsAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

func (b *ResourceGroupsAccessBuilder) WithCheckExistence() *ResourceGroupsAccessBuilder {
	b.rgServer.CheckExistence = func(ctx context.Context, resourceGroupName string, options *armresources.ResourceGroupsClientCheckExistenceOptions) (resp azfake.Responder[armresources.ResourceGroupsClientCheckExistenceResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.Simulate(ctx, resourceGroupName, resourceGroupName, test.AccessMethodCheckExistence)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.rg != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceGroupNotFound))
			return
		}
		rgResponse := armresources.ResourceGroupsClientCheckExistenceResponse{Success: true}
		resp.SetResponse(http.StatusNoContent, rgResponse, nil)
		return
	}
	return b
}

func (b *ResourceGroupsAccessBuilder) Build() (*armresources.ResourceGroupsClient, error) {
	b.WithCheckExistence()
	return armresources.NewResourceGroupsClient(
		test.SubscriptionID,
		azfake.NewTokenCredential(),
		&arm.ClientOptions{
			ClientOptions: policy.ClientOptions{
				Transport: fakearmresources.NewResourceGroupsServerTransport(&b.rgServer),
			},
		},
	)
}
