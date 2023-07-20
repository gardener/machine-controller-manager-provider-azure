package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	fakenetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

type NICAccessBuilder struct {
	clusterState    *ClusterState
	nicServer       fakenetwork.InterfacesServer
	apiBehaviorSpec *APIBehaviorSpec
}

func (b *NICAccessBuilder) WithClusterState(clusterState *ClusterState) *NICAccessBuilder {
	b.clusterState = clusterState
	return b
}

func (b *NICAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *NICAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

func (b *NICAccessBuilder) WithGet() *NICAccessBuilder {
	b.nicServer.Get = func(ctx context.Context, resourceGroupName string, nicName string, options *armnetwork.InterfacesClientGetOptions) (resp azfake.Responder[armnetwork.InterfacesClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.Simulate(ctx, resourceGroupName, nicName, test.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ResourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceGroupNotFound))
			return
		}
		nic := b.clusterState.GetNIC(nicName)
		if nic == nil {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceNotFound))
			return
		}
		nicResponse := armnetwork.InterfacesClientGetResponse{Interface: *nic}
		resp.SetResponse(http.StatusOK, nicResponse, nil)
		return
	}
	return b
}

func (b *NICAccessBuilder) WithBeginDelete() *NICAccessBuilder {
	b.nicServer.BeginDelete = func(ctx context.Context, resourceGroupName string, nicName string, options *armnetwork.InterfacesClientBeginDeleteOptions) (resp azfake.PollerResponder[armnetwork.InterfacesClientDeleteResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.Simulate(ctx, resourceGroupName, nicName, test.AccessMethodBeginDelete)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ResourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceGroupNotFound))
			return
		}
		// Azure API NIC deletion does not fail if the NIC does not exist. It still returns 200 Ok.
		nic := b.clusterState.GetNIC(nicName)
		if nic != nil && nic.Properties.VirtualMachine != nil && !utils.IsNilOrEmptyStringPtr(nic.Properties.VirtualMachine.ID) {
			errResp.SetError(ConflictErr(test.ErrorOperationNotAllowed))
			return
		}
		b.clusterState.DeleteNIC(nicName)
		resp.SetTerminalResponse(200, armnetwork.InterfacesClientDeleteResponse{}, nil)
		return
	}
	return b
}

func (b *NICAccessBuilder) Build() (*armnetwork.InterfacesClient, error) {
	b.WithGet().WithBeginDelete()
	return armnetwork.NewInterfacesClient(test.SubscriptionID, azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakenetwork.NewInterfacesServerTransport(&b.nicServer),
		},
	})
}
