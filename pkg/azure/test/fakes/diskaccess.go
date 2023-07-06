package fakes

import (
	"context"
	"net/http"

	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
)

type DiskAccessBuilder struct {
	resourceGroup string
	existingDisks map[string]armcompute.Disk
	diskServer    fakecompute.DisksServer
}

func (b *DiskAccessBuilder) WithExistingDisks(disks []armcompute.Disk) *DiskAccessBuilder {
	if b.existingDisks == nil {
		b.existingDisks = make(map[string]armcompute.Disk)
	}
	for _, disk := range disks {
		b.existingDisks[*disk.Name] = disk
	}
	return b
}

func (b *DiskAccessBuilder) WithGet(apiBehaviorOpts *APIBehaviorOptions) *DiskAccessBuilder {
	b.diskServer.Get = func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (resp azfake.Responder[armcompute.DisksClientGetResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.resourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceGroupNotFound))
			return
		}
		disk, exists := b.existingDisks[diskName]
		if !exists {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceNotFound))
			return
		}
		diskResponse := armcompute.DisksClientGetResponse{Disk: disk}
		resp.SetResponse(http.StatusOK, diskResponse, nil)
		return
	}
	return b
}

func (b *DiskAccessBuilder) WithBeginDelete(apiBehaviorOpts *APIBehaviorOptions) *DiskAccessBuilder {
	b.diskServer.BeginDelete = func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientBeginDeleteOptions) (resp azfake.PollerResponder[armcompute.DisksClientDeleteResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.resourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(ErrorCodeResourceGroupNotFound))
			return
		}

		// Azure API Disk deletion does not fail if the Disk does not exist. It still returns 200 Ok.
		delete(b.existingDisks, diskName)
		resp.SetTerminalResponse(http.StatusOK, armcompute.DisksClientDeleteResponse{}, nil)
		return
	}
	return b
}
