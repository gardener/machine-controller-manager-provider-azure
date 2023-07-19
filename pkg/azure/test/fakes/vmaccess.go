package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

type VMAccessBuilder struct {
	clusterState *ClusterState
	vmServer     fakecompute.VirtualMachinesServer
}

func (b *VMAccessBuilder) WithClusterState(clusterState *ClusterState) *VMAccessBuilder {
	b.clusterState = clusterState
	return b
}

func (b *VMAccessBuilder) WithDefaultAPIBehavior() *VMAccessBuilder {
	return b.WithGet(nil).
		WithBeginDelete(nil).
		WithBeginUpdate(nil)
}

func (b *VMAccessBuilder) WithGet(apiBehaviorOpts *APIBehaviorOptions) *VMAccessBuilder {
	b.vmServer.Get = func(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachinesClientGetResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.clusterState.ResourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceGroupNotFound))
			return
		}
		machineResources, existing := b.clusterState.MachineResourcesMap[vmName]
		if !existing || machineResources.VM == nil {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceNotFound))
			return
		}
		vmResp := armcompute.VirtualMachinesClientGetResponse{VirtualMachine: *machineResources.VM}
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
		if b.clusterState.ResourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceGroupNotFound))
			return
		}

		b.clusterState.DeleteVM(vmName)
		// Azure API VM deletion does not fail if the VM does not exist. It still returns 200 Ok.
		resp.SetTerminalResponse(200, armcompute.VirtualMachinesClientDeleteResponse{}, nil)
		return
	}
	return b
}

func (b *VMAccessBuilder) WithBeginUpdate(apiBehaviorOpts *APIBehaviorOptions) *VMAccessBuilder {
	b.vmServer.BeginUpdate = func(ctx context.Context, resourceGroupName string, vmName string, updateParams armcompute.VirtualMachineUpdate, options *armcompute.VirtualMachinesClientBeginUpdateOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientUpdateResponse], errResp azfake.ErrorResponder) {
		if apiBehaviorOpts != nil && apiBehaviorOpts.TimeoutAfter != nil {
			errResp.SetError(ContextTimeoutError(ctx, *apiBehaviorOpts.TimeoutAfter))
			return
		}
		if b.clusterState.ResourceGroup != resourceGroupName {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceGroupNotFound))
			return
		}
		machineResources, existing := b.clusterState.MachineResourcesMap[vmName]
		if !existing || machineResources.VM == nil {
			errResp.SetError(ResourceNotFoundErr(test.ErrorCodeResourceNotFound))
			return
		}

		// NOTE: Currently we are only using this API to set cascade delete option for NIC and Disks.
		// So to avoid complexity, we will restrict it to only updating cascade delete options only.
		// If in future the usage changes then changes should also be done here to reflect that.
		b.updateNICCascadeDeleteOption(vmName, updateParams.Properties.NetworkProfile)
		b.updateOSDiskCascadeDeleteOption(vmName, updateParams.Properties.StorageProfile)
		b.updatedDataDisksCascadeDeleteOption(vmName, updateParams.Properties.StorageProfile)

		m := b.clusterState.MachineResourcesMap[vmName]
		resp.SetTerminalResponse(200, armcompute.VirtualMachinesClientUpdateResponse{VirtualMachine: *m.VM}, nil)
		return
	}
	return b
}

func (b *VMAccessBuilder) updateNICCascadeDeleteOption(vmName string, nwProfile *armcompute.NetworkProfile) {
	var deleteOpt *armcompute.DeleteOptions
	if nwProfile != nil {
		nwInterfaces := nwProfile.NetworkInterfaces
		if !utils.IsSliceNilOrEmpty(nwInterfaces) {
			properties := nwInterfaces[0].Properties
			if properties != nil {
				deleteOpt = properties.DeleteOption
			}
		}
		m := b.clusterState.MachineResourcesMap[vmName]
		m.cascadeDeleteOpts.NIC = deleteOpt
	}
}

func (b *VMAccessBuilder) updateOSDiskCascadeDeleteOption(vmName string, storageProfile *armcompute.StorageProfile) {
	var deleteOpt *armcompute.DiskDeleteOptionTypes
	if storageProfile != nil {
		osDisk := storageProfile.OSDisk
		if osDisk != nil {
			deleteOpt = osDisk.DeleteOption
			m := b.clusterState.MachineResourcesMap[vmName]
			m.cascadeDeleteOpts.OSDisk = deleteOpt
		}
	}
}

// updatedDataDisksCascadeDeleteOption updates the cascade delete option for data disks that are associated to a VM.
// It is assumed that consumer will uniformly set delete option for all data disks. This is the only case we also support
// in gardener and to ensure simplicity of tests we will not support a case where different data disk can have different delete options.
// So the implementation only takes the first DataDisk and assumes the delete option for rest of them as well.
func (b *VMAccessBuilder) updatedDataDisksCascadeDeleteOption(vmName string, storageProfile *armcompute.StorageProfile) {
	var deleteOpt *armcompute.DiskDeleteOptionTypes
	if storageProfile != nil {
		dataDisks := storageProfile.DataDisks
		if !utils.IsSliceNilOrEmpty(dataDisks) {
			deleteOpt = dataDisks[0].DeleteOption
			m := b.clusterState.MachineResourcesMap[vmName]
			m.cascadeDeleteOpts.DataDisk = deleteOpt
		}
	}
}

func (b *VMAccessBuilder) Build() (*armcompute.VirtualMachinesClient, error) {
	return armcompute.NewVirtualMachinesClient(test.SubscriptionID, azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fakecompute.NewVirtualMachinesServerTransport(&b.vmServer),
		},
	})
}
