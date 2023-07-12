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
)

// Default timeouts for all async operations - Create/Delete/Update
const (
	defaultDeleteVMTimeout = 15 * time.Minute
	defaultCreateVMTimeout = 15 * time.Minute
	defaultUpdateVMTimeout = 10 * time.Minute
)

// GetVirtualMachine gets a VirtualMachine for the given vm name and resource group.
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
func DeleteVirtualMachine(ctx context.Context, vmClient *armcompute.VirtualMachinesClient, resourceGroup, vmName string) (err error) {
	defer instrument.RecordAzAPIMetric(err, vmDeleteServiceLabel, time.Now())
	delCtx, cancelFn := context.WithTimeout(ctx, defaultDeleteVMTimeout)
	defer cancelFn()
	poller, err := vmClient.BeginDelete(delCtx, resourceGroup, vmName, nil)
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

// SetCascadeDeleteForNICsAndDisks sets cascade deletion for NICs and Disks (OSDisk and DataDisks) associated to passed virtual machine.
func SetCascadeDeleteForNICsAndDisks(ctx context.Context, vmClient *armcompute.VirtualMachinesClient, resourceGroup string, vm *armcompute.VirtualMachine) (err error) {
	defer instrument.RecordAzAPIMetric(err, vmUpdateServiceLabel, time.Now())
	vmUpdateParams := createVMUpdateParamsForCascadeDeleteOptions(vm) // TODO: Rename this method it returns a VM not "params"
	if vmUpdateParams == nil {
		klog.Infof("All configured NICs, OSDisk and DataDisks have cascade delete already set. Skipping update of VM: [ResourceGroup: %s, Name: %s]", resourceGroup, *vm.Name)
		return
	}
	updCtx, cancelFn := context.WithTimeout(ctx, defaultUpdateVMTimeout)
	defer cancelFn()
	poller, err := vmClient.BeginUpdate(updCtx, resourceGroup, *vm.Name, *vmUpdateParams, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger update of VM [ResourceGroup: %s, VMName: %s]", resourceGroup, *vm.Name)
		return
	}
	pollResp, err := poller.PollUntilDone(updCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for update of VM: %s for ResourceGroup: %s", *vm.Name, resourceGroup)
		return
	}

	_, err = pollResp.MarshalJSON()
	if err != nil {
		klog.V(4).Infof("failed to marshal VM update response JSON for [ResourceGroup: %s, VMName: %s], Err: %s", resourceGroup, *vm.Name, err.Error())
	}

	return
}

// createVMUpdateParamsForCascadeDeleteOptions creates armcompute.VirtualMachineUpdate with delta changes to
// delete option for associated NICs and Disks of a given virtual machine.
func createVMUpdateParamsForCascadeDeleteOptions(vm *armcompute.VirtualMachine) *armcompute.VirtualMachineUpdate {
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
