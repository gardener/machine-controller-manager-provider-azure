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
	rg       string
	rgServer fakearmresources.ResourceGroupsServer
}

func (b *ResourceGroupsAccessBuilder) WithCheckExistence(apiBehaviorOpts *APIBehaviorOptions) *ResourceGroupsAccessBuilder {
	b.rgServer.CheckExistence = func(ctx context.Context, resourceGroupName string, options *armresources.ResourceGroupsClientCheckExistenceOptions) (resp azfake.Responder[armresources.ResourceGroupsClientCheckExistenceResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
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
