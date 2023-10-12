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

package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

// labels used for recording prometheus metrics
const (
	vmGetServiceLabel    = "virtual_machine_get"
	vmUpdateServiceLabel = "virtual_machine_update"
	vmDeleteServiceLabel = "virtual_machine_delete"
	vmCreateServiceLabel = "virtual_machine_create"
)

// Default timeouts for all async operations - Create/Delete/Update
const (
	defaultDeleteVMTimeout = 15 * time.Minute
	defaultCreateVMTimeout = 15 * time.Minute
	defaultUpdateVMTimeout = 10 * time.Minute
)

// GetVirtualMachine gets a VirtualMachine for the given vm name and resource group.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func GetVirtualMachine(ctx context.Context, vmClient *armcompute.VirtualMachinesClient, resourceGroup, vmName string) (vm *armcompute.VirtualMachine, err error) {
	var getResp armcompute.VirtualMachinesClientGetResponse
	defer instrument.RecordAzAPIMetric(err, vmGetServiceLabel, time.Now())
	getResp, err = vmClient.Get(ctx, resourceGroup, vmName, nil)
	if err != nil {
		if errors.IsNotFoundAzAPIError(err) {
			return nil, nil
		}
		return
	}
	vm = &getResp.VirtualMachine
	return
}

// DeleteVirtualMachine deletes the Virtual Machine with the give name and belonging to the passed in resource group.
// If cascade delete is set for associated NICs and Disks then these resources will also be deleted along with the VM.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func DeleteVirtualMachine(ctx context.Context, vmAccess *armcompute.VirtualMachinesClient, resourceGroup, vmName string) (err error) {
	defer instrument.RecordAzAPIMetric(err, vmDeleteServiceLabel, time.Now())
	delCtx, cancelFn := context.WithTimeout(ctx, defaultDeleteVMTimeout)
	defer cancelFn()
	poller, err := vmAccess.BeginDelete(delCtx, resourceGroup, vmName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger delete of VM [ResourceGroup: %s, VMName: %s]", resourceGroup, vmName)
		return
	}
	_, err = poller.PollUntilDone(delCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for delete of VM: %s for ResourceGroup: %s", vmName, resourceGroup)
		return
	}
	return
}

// CreateVirtualMachine creates a Virtual Machine given a resourceGroup and virtual machine creation parameters.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func CreateVirtualMachine(ctx context.Context, vmAccess *armcompute.VirtualMachinesClient, resourceGroup string, vmCreationParams armcompute.VirtualMachine) (vm *armcompute.VirtualMachine, err error) {
	defer instrument.RecordAzAPIMetric(err, vmCreateServiceLabel, time.Now())
	createCtx, cancelFn := context.WithTimeout(ctx, defaultCreateVMTimeout)
	defer cancelFn()
	vmName := *vmCreationParams.Name
	poller, err := vmAccess.BeginCreateOrUpdate(createCtx, resourceGroup, vmName, vmCreationParams, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger create of VM [ResourceGroup: %s, VMName: %s]", resourceGroup, vmName)
		return
	}
	createResp, err := poller.PollUntilDone(createCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for create of VM: %s for ResourceGroup: %s", vmName, resourceGroup)
		return
	}
	vm = &createResp.VirtualMachine
	return
}

// SetCascadeDeleteForNICsAndDisks sets cascade deletion for NICs and Disks (OSDisk and DataDisks) associated to passed virtual machine.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func SetCascadeDeleteForNICsAndDisks(ctx context.Context, vmClient *armcompute.VirtualMachinesClient, resourceGroup string, vm *armcompute.VirtualMachine) (err error) {
	defer instrument.RecordAzAPIMetric(err, vmUpdateServiceLabel, time.Now())
	vmUpdateDesc := createVirtualMachineUpdateDescription(vm) // TODO: Rename this method it returns a VM not "params"
	if vmUpdateDesc == nil {
		klog.Infof("All configured NICs, OSDisk and DataDisks have cascade delete already set. Skipping update of VM: [ResourceGroup: %s, Name: %s]", resourceGroup, *vm.Name)
		return
	}
	updCtx, cancelFn := context.WithTimeout(ctx, defaultUpdateVMTimeout)
	defer cancelFn()
	poller, err := vmClient.BeginUpdate(updCtx, resourceGroup, *vm.Name, *vmUpdateDesc, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger update of VM [ResourceGroup: %s, VMName: %s]", resourceGroup, *vm.Name)
		return
	}
	_, err = poller.PollUntilDone(updCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for update of VM: %s for ResourceGroup: %s", *vm.Name, resourceGroup)
		return
	}

	return
}

// createVirtualMachineUpdateDescription creates armcompute.VirtualMachineUpdate with delta changes to
// delete option for associated NICs and Disks of a given virtual machine.
func createVirtualMachineUpdateDescription(vm *armcompute.VirtualMachine) *armcompute.VirtualMachineUpdate {
	var (
		vmUpdateParams              = armcompute.VirtualMachineUpdate{Properties: &armcompute.VirtualMachineProperties{}}
		cascadeDeleteChangesPending bool
	)

	updatedNicRefs := getNetworkInterfaceReferencesToUpdate(vm.Properties.NetworkProfile)
	if !utils.IsSliceNilOrEmpty(updatedNicRefs) {
		cascadeDeleteChangesPending = true
		vmUpdateParams.Properties.NetworkProfile = &armcompute.NetworkProfile{
			NetworkInterfaces: updatedNicRefs,
		}
	}

	vmUpdateParams.Properties.StorageProfile = &armcompute.StorageProfile{}
	updatedOSDisk := getOSDiskToUpdate(vm.Properties.StorageProfile)
	if updatedOSDisk != nil {
		cascadeDeleteChangesPending = true
		vmUpdateParams.Properties.StorageProfile.OSDisk = updatedOSDisk
	}

	updatedDataDisks := getDataDisksToUpdate(vm.Properties.StorageProfile)
	if !utils.IsSliceNilOrEmpty(updatedDataDisks) {
		cascadeDeleteChangesPending = true
		vmUpdateParams.Properties.StorageProfile.DataDisks = updatedDataDisks
	}

	if !cascadeDeleteChangesPending {
		return nil
	}
	return &vmUpdateParams
}

func getNetworkInterfaceReferencesToUpdate(networkProfile *armcompute.NetworkProfile) []*armcompute.NetworkInterfaceReference {
	if networkProfile == nil || utils.IsSliceNilOrEmpty(networkProfile.NetworkInterfaces) {
		return nil
	}
	updatedNicRefs := make([]*armcompute.NetworkInterfaceReference, 0, len(networkProfile.NetworkInterfaces))
	for _, nicRef := range networkProfile.NetworkInterfaces {
		updatedNicRef := &armcompute.NetworkInterfaceReference{ID: nicRef.ID}
		if !isNicCascadeDeleteSet(nicRef) {
			if updatedNicRef.Properties == nil {
				updatedNicRef.Properties = &armcompute.NetworkInterfaceReferenceProperties{}
			}
			updatedNicRef.Properties.DeleteOption = to.Ptr(armcompute.DeleteOptionsDelete)
			updatedNicRefs = append(updatedNicRefs, updatedNicRef)
		}
	}
	return updatedNicRefs
}

func isNicCascadeDeleteSet(nicRef *armcompute.NetworkInterfaceReference) bool {
	if nicRef.Properties == nil {
		return false
	}
	deleteOption := nicRef.Properties.DeleteOption
	return deleteOption != nil && *deleteOption == armcompute.DeleteOptionsDelete
}

func getOSDiskToUpdate(storageProfile *armcompute.StorageProfile) *armcompute.OSDisk {
	var updatedOSDisk *armcompute.OSDisk
	if storageProfile != nil && storageProfile.OSDisk != nil {
		existingOSDisk := storageProfile.OSDisk
		existingDeleteOption := existingOSDisk.DeleteOption
		if existingDeleteOption == nil || *existingDeleteOption != armcompute.DiskDeleteOptionTypesDelete {
			updatedOSDisk = &armcompute.OSDisk{
				Name:         existingOSDisk.Name,
				DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
			}
		}
	}
	return updatedOSDisk
}

func getDataDisksToUpdate(storageProfile *armcompute.StorageProfile) []*armcompute.DataDisk {
	var updatedDataDisks []*armcompute.DataDisk
	if storageProfile != nil && !utils.IsSliceNilOrEmpty(storageProfile.DataDisks) {
		updatedDataDisks = make([]*armcompute.DataDisk, 0, len(storageProfile.DataDisks))
		for _, dataDisk := range storageProfile.DataDisks {
			if dataDisk.DeleteOption == nil || *dataDisk.DeleteOption != armcompute.DiskDeleteOptionTypesDelete {
				updatedDataDisk := &armcompute.DataDisk{
					Lun:          dataDisk.Lun,
					DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
					Name:         dataDisk.Name,
				}
				updatedDataDisks = append(updatedDataDisks, updatedDataDisk)
			}
		}
	}
	return updatedDataDisks
}
