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

func (b *VMAccessBuilder) WithGet(apiBehaviorOpts *APIBehaviorOptions) *VMAccessBuilder {
	b.vmServer.Get = func(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachinesClientGetResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
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

func (b *VMAccessBuilder) WithBeginDelete(apiBehaviorOpts *APIBehaviorOptions) *VMAccessBuilder {
	b.vmServer.BeginDelete = func(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientBeginDeleteOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientDeleteResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.resourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceGroupNotFound))
			return
		}

		// Azure API VM deletion does not fail if the VM does not exist. It still returns 200 Ok.
		delete(b.existingVms, vmName)
		resp.SetTerminalResponse(200, armcompute.VirtualMachinesClientDeleteResponse{}, nil)
		return
	}
	return b
}

func (b *VMAccessBuilder) WithBeginUpdate(apiBehaviorOpts *APIBehaviorOptions) *VMAccessBuilder {
	b.vmServer.BeginUpdate = func(ctx context.Context, resourceGroupName string, vmName string, parameters armcompute.VirtualMachineUpdate, options *armcompute.VirtualMachinesClientBeginUpdateOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientUpdateResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.resourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(ErrorCodePatchResourceNotFound))
			return
		}

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
