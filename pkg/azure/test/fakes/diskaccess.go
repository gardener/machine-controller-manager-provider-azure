package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

type DiskAccessBuilder struct {
	clusterState *ClusterState
	diskServer   fakecompute.DisksServer
}

func (b *DiskAccessBuilder) WithClusterState(clusterState *ClusterState) *DiskAccessBuilder {
	b.clusterState = clusterState
	return b
}

func (b *DiskAccessBuilder) WithDefaultAPIBehavior() *DiskAccessBuilder {
	return b.WithGet(nil).WithBeginDelete(nil)
}

func (b *DiskAccessBuilder) WithGet(apiBehaviorOpts *APIBehaviorOptions) *DiskAccessBuilder {
	b.diskServer.Get = func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (resp azfake.Responder[armcompute.DisksClientGetResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.clusterState.ResourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceGroupNotFound))
			return
		}
		disk := b.clusterState.GetDisk(diskName)
		if disk == nil {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceNotFound))
			return
		}
		diskResponse := armcompute.DisksClientGetResponse{Disk: *disk}
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
		if b.clusterState.ResourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceGroupNotFound))
			return
		}

		// Azure API Disk deletion does not fail if the Disk does not exist. It still returns 200 Ok.
		disk := b.clusterState.GetDisk(diskName)
		if disk != nil && !utils.IsNilOrEmptyStringPtr(disk.ManagedBy) {
			errResp.SetError(ConflictErr(test.ErrorOperationNotAllowed))
			return
		}
		b.clusterState.DeleteDisk(diskName)
		resp.SetTerminalResponse(http.StatusOK, armcompute.DisksClientDeleteResponse{}, nil)
		return
	}
	return b
}

func (b *DiskAccessBuilder) Build() (*armcompute.DisksClient, error) {
	return armcompute.NewDisksClient(test.SubscriptionID, azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakecompute.NewDisksServerTransport(&b.diskServer),
		},
	})
}
