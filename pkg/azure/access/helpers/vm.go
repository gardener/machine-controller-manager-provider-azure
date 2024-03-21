// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"k8s.io/klog/v2"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
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
	//defaultUpdateVMTimeout is the timeout required to complete an update of a VM. It is currently
	// seen that update is relatively faster and therefore a lower timeout has been kept. This could
	// be changed in the future depending on the metrics that we record and observe.
	defaultUpdateVMTimeout = 10 * time.Minute
)

// GetVirtualMachine gets a VirtualMachine for the given vm name and resource group.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func GetVirtualMachine(ctx context.Context, vmClient *armcompute.VirtualMachinesClient, resourceGroup, vmName string) (vm *armcompute.VirtualMachine, err error) {
	var getResp armcompute.VirtualMachinesClientGetResponse
	defer instrument.AZAPIMetricRecorderFn(vmGetServiceLabel, &err)()

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
	defer instrument.AZAPIMetricRecorderFn(vmDeleteServiceLabel, &err)()

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
	klog.Infof("Successfully deleted VM: %s, for ResourceGroup: %s", vmName, resourceGroup)
	return
}

// CreateVirtualMachine creates a Virtual Machine given a resourceGroup and virtual machine creation parameters.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func CreateVirtualMachine(ctx context.Context, vmAccess *armcompute.VirtualMachinesClient, resourceGroup string, vmCreationParams armcompute.VirtualMachine) (vm *armcompute.VirtualMachine, err error) {
	defer instrument.AZAPIMetricRecorderFn(vmCreateServiceLabel, &err)()

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
func SetCascadeDeleteForNICsAndDisks(ctx context.Context, vmClient *armcompute.VirtualMachinesClient, resourceGroup string, vmName string, vmUpdateParams *armcompute.VirtualMachineUpdate) (err error) {
	defer instrument.AZAPIMetricRecorderFn(vmUpdateServiceLabel, &err)()

	updCtx, cancelFn := context.WithTimeout(ctx, defaultUpdateVMTimeout)
	defer cancelFn()
	poller, err := vmClient.BeginUpdate(updCtx, resourceGroup, vmName, *vmUpdateParams, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger update of VM [ResourceGroup: %s, VMName: %s]", resourceGroup, vmName)
		return
	}
	_, err = poller.PollUntilDone(updCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for update of VM: %s for ResourceGroup: %s", vmName, resourceGroup)
		return
	}
	return
}
