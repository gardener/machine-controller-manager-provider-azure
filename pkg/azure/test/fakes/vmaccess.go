package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
)

type VMAccessBuilder struct {
	resourceGroup string
	existingVms   map[string]armcompute.VirtualMachine
	vmServer      fakecompute.VirtualMachinesServer
}

func (b *VMAccessBuilder) WithExistingVMs(vms []armcompute.VirtualMachine) *VMAccessBuilder {
	if b.existingVms == nil {
		b.existingVms = make(map[string]armcompute.VirtualMachine, len(vms))
	}
	for _, v := range vms {
		b.existingVms[*v.Name] = v
	}
	return b
}

func (b *VMAccessBuilder) WithGet(apiBehavior *APIBehaviorOptions) *VMAccessBuilder {
	b.vmServer.Get = func(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachinesClientGetResponse], errResp azfake.ErrorResponder) {
		if apiBehavior != nil && apiBehavior.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehavior.TimeoutAfter))
			return
		}
		if b.resourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceGroupNotFound))
			return
		}
		vm, existingVM := b.existingVms[vmName]
		if !existingVM {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceNotFound))
			return
		}
		vmResp := armcompute.VirtualMachinesClientGetResponse{VirtualMachine: vm}
		resp.SetResponse(http.StatusOK, vmResp, nil)
		return
	}
	return b
}

func (b *VMAccessBuilder) WithBeginDelete(apiBehavior *APIBehaviorOptions) *VMAccessBuilder {
	b.vmServer.BeginDelete = func(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientBeginDeleteOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientDeleteResponse], errResp azfake.ErrorResponder) {
		if apiBehavior != nil && apiBehavior.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehavior.TimeoutAfter))
			return
		}
		if b.resourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceGroupNotFound))
			return
		}
		delete(b.existingVms, vmName)
		resp.SetTerminalResponse(200, armcompute.VirtualMachinesClientDeleteResponse{}, nil)
		return
	}
	return b
}

func (b *VMAccessBuilder) Build() (*armcompute.VirtualMachinesClient, error) {
	return armcompute.NewVirtualMachinesClient(TestSubscriptionID, azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakecompute.NewVirtualMachinesServerTransport(&b.vmServer),
		},
	})
}
