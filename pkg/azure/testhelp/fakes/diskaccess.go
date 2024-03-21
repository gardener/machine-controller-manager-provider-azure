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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

// DiskAccessBuilder is a builder for armcompute.DisksClient.
type DiskAccessBuilder struct {
	clusterState    *ClusterState
	server          fakecompute.DisksServer
	apiBehaviorSpec *APIBehaviorSpec
}

// WithClusterState initializes builder with a ClusterState.
func (b *DiskAccessBuilder) WithClusterState(clusterState *ClusterState) *DiskAccessBuilder {
	b.clusterState = clusterState
	return b
}

// WithAPIBehaviorSpec initializes the builder with a APIBehaviorSpec.
func (b *DiskAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *DiskAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

// withGet implements the Get method of armcompute.DisksClient and initializes the backing fake server's Get method with the anonymous function implementation.
func (b *DiskAccessBuilder) withGet() *DiskAccessBuilder {
	b.server.Get = func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (resp azfake.Responder[armcompute.DisksClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, diskName, testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}
		disk := b.clusterState.GetDisk(diskName)
		if disk == nil {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceNotFound))
			return
		}
		diskResponse := armcompute.DisksClientGetResponse{Disk: *disk}
		resp.SetResponse(http.StatusOK, diskResponse, nil)
		return
	}
	return b
}

// withBeginDelete implements the BeingDelete method of armcompute.DisksClient and initializes the backing fake server's BeginDelete method with the anonymous function implementation.
func (b *DiskAccessBuilder) withBeginDelete() *DiskAccessBuilder {
	b.server.BeginDelete = func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientBeginDeleteOptions) (resp azfake.PollerResponder[armcompute.DisksClientDeleteResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, diskName, testhelp.AccessMethodBeginDelete)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}

		// Azure API Disk deletion does not fail if the Disk does not exist. It still returns 200 Ok.
		disk := b.clusterState.GetDisk(diskName)
		if disk != nil && !utils.IsNilOrEmptyStringPtr(disk.ManagedBy) {
			errResp.SetError(testhelp.ConflictErr(testhelp.ErrorCodeOperationNotAllowed))
			return
		}
		b.clusterState.DeleteDisk(diskName)
		resp.SetTerminalResponse(http.StatusOK, armcompute.DisksClientDeleteResponse{}, nil)
		return
	}
	return b
}

// Build builds the armcompute.DiskClient.
func (b *DiskAccessBuilder) Build() (*armcompute.DisksClient, error) {
	b.withGet().withBeginDelete()
	return armcompute.NewDisksClient(testhelp.SubscriptionID, &azfake.TokenCredential{}, &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakecompute.NewDisksServerTransport(&b.server),
		},
	})
}
