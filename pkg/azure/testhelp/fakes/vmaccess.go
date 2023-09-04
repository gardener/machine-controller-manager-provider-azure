// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

// VMAccessBuilder is a builder for VM access.
type VMAccessBuilder struct {
	clusterState    *ClusterState
	server          fakecompute.VirtualMachinesServer
	apiBehaviorSpec *APIBehaviorSpec
}

// WithClusterState initializes builder with a ClusterState.
func (b *VMAccessBuilder) WithClusterState(clusterState *ClusterState) *VMAccessBuilder {
	b.clusterState = clusterState
	return b
}

// WithAPIBehaviorSpec initializes the builder with a APIBehaviorSpec.
func (b *VMAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *VMAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

// withGet implements the Get method of armcompute.VirtualMachinesClient and initializes the backing fake server's Get method with the anonymous function implementation.
func (b *VMAccessBuilder) withGet() *VMAccessBuilder {
	b.server.Get = func(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachinesClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, vmName, testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}
		machineResources, existing := b.clusterState.MachineResourcesMap[vmName]
		if !existing || machineResources.VM == nil {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceNotFound))
			return
		}
		vmResp := armcompute.VirtualMachinesClientGetResponse{VirtualMachine: *machineResources.VM}
		resp.SetResponse(http.StatusOK, vmResp, nil)
		return
	}
	return b
}

// withBeginDelete implements the BeingDelete method of armcompute.VirtualMachinesClient and initializes the backing fake server's BeginDelete method with the anonymous function implementation.
func (b *VMAccessBuilder) withBeginDelete() *VMAccessBuilder {
	b.server.BeginDelete = func(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientBeginDeleteOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientDeleteResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, vmName, testhelp.AccessMethodBeginDelete)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}

		b.clusterState.DeleteVM(vmName)
		// Azure API VM deletion does not fail if the VM does not exist. It still returns 200 Ok.
		resp.SetTerminalResponse(200, armcompute.VirtualMachinesClientDeleteResponse{}, nil)
		return
	}
	return b
}

// withBeginCreateOrUpdate implements the BeginCreateOrUpdate method of armcompute.VirtualMachinesClient and initializes the backing fake server's BeginCreateOrUpdate method with the anonymous function implementation.
func (b *VMAccessBuilder) withBeginCreateOrUpdate() *VMAccessBuilder {
	b.server.BeginCreateOrUpdate = func(ctx context.Context, resourceGroupName string, vmName string, parameters armcompute.VirtualMachine, options *armcompute.VirtualMachinesClientBeginCreateOrUpdateOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientCreateOrUpdateResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, vmName, testhelp.AccessMethodBeginCreateOrUpdate)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}

		vm := b.clusterState.CreateVM(resourceGroupName, parameters)
		resp.SetTerminalResponse(http.StatusOK, armcompute.VirtualMachinesClientCreateOrUpdateResponse{VirtualMachine: *vm}, nil)
		return
	}
	return b
}

// withBeginUpdate implements the BeingUpdate method of armcompute.VirtualMachinesClient and initializes the backing fake server's BeginUpdate method with the anonymous function implementation.
func (b *VMAccessBuilder) withBeginUpdate() *VMAccessBuilder {
	b.server.BeginUpdate = func(ctx context.Context, resourceGroupName string, vmName string, updateParams armcompute.VirtualMachineUpdate, options *armcompute.VirtualMachinesClientBeginUpdateOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientUpdateResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, vmName, testhelp.AccessMethodBeginUpdate)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}
		machineResources, existing := b.clusterState.MachineResourcesMap[vmName]
		if !existing || machineResources.VM == nil {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceNotFound))
			return
		}

		// NOTE: Currently we are only using update API to set cascade delete option for NIC and Disks.
		// So to avoid complexity, we will restrict it to only updating cascade delete options only.
		// If in future the usage changes then changes should also be done here to reflect that.
		b.updateNICCascadeDeleteOption(vmName, updateParams.Properties.NetworkProfile)
		b.updateOSDiskCascadeDeleteOption(vmName, updateParams.Properties.StorageProfile)
		b.updatedDataDisksCascadeDeleteOption(vmName, updateParams.Properties.StorageProfile)

		// Get the updated VM
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
		m.UpdateNICDeleteOpt(deleteOpt)
		//b.clusterState.MachineResourcesMap[vmName] = m
	}
}

func (b *VMAccessBuilder) updateOSDiskCascadeDeleteOption(vmName string, storageProfile *armcompute.StorageProfile) {
	var deleteOpt *armcompute.DiskDeleteOptionTypes
	if storageProfile != nil {
		osDisk := storageProfile.OSDisk
		if osDisk != nil {
			deleteOpt = osDisk.DeleteOption
			m := b.clusterState.MachineResourcesMap[vmName]
			m.UpdateOSDiskDeleteOpt(deleteOpt)
			//b.clusterState.MachineResourcesMap[vmName] = m
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
			m.UpdateDataDisksDeleteOpt(deleteOpt)
			//b.clusterState.MachineResourcesMap[vmName] = m
		}
	}
}

// Build builds armcompute.VirtualMachinesClient.
func (b *VMAccessBuilder) Build() (*armcompute.VirtualMachinesClient, error) {
	b.withGet().withBeginDelete().withBeginUpdate().withBeginCreateOrUpdate()
	return armcompute.NewVirtualMachinesClient(testhelp.SubscriptionID, azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fakecompute.NewVirtualMachinesServerTransport(&b.server),
		},
	})
}
