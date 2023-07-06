package fakes

import (
	"context"
	"net/http"

	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	fakenetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3/fake"
)

type NICAccessBuilder struct {
	resourceGroup string
	existingNICs  map[string]armnetwork.Interface
	nicServer     fakenetwork.InterfacesServer
}

func (b *NICAccessBuilder) WithExistingNICs(nics []armnetwork.Interface) *NICAccessBuilder {
	if b.existingNICs == nil {
		b.existingNICs = make(map[string]armnetwork.Interface)
	}
	for _, nic := range nics {
		b.existingNICs[*nic.Name] = nic
	}
	return b
}

func (b *NICAccessBuilder) WithGet(apiBehaviorOpts *APIBehaviorOptions) *NICAccessBuilder {
	b.nicServer.Get = func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientGetOptions) (resp azfake.Responder[armnetwork.InterfacesClientGetResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.resourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceGroupNotFound))
			return
		}
		nic, exists := b.existingNICs[networkInterfaceName]
		if !exists {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceNotFound))
			return
		}
		nicResponse := armnetwork.InterfacesClientGetResponse{Interface: nic}
		resp.SetResponse(http.StatusOK, nicResponse, nil)
		return
	}
	return b
}

func (b *NICAccessBuilder) WithBeginDelete(apiBehaviorOpts *APIBehaviorOptions) *NICAccessBuilder {
	b.nicServer.BeginDelete = func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginDeleteOptions) (resp azfake.PollerResponder[armnetwork.InterfacesClientDeleteResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.resourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceGroupNotFound))
			return
		}
		// Azure API NIC deletion does not fail if the NIC does not exist. It still returns 200 Ok.
		delete(b.existingNICs, networkInterfaceName)
		resp.SetTerminalResponse(200, armnetwork.InterfacesClientDeleteResponse{}, nil)
		return
	}
	return b
}
