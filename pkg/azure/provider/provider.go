// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	clienthelpers "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/provider/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

const (
	createMachineOperationLabel    = "create_machine"
	deleteMachineOperationLabel    = "delete_machine"
	listMachinesOperationLabel     = "list_machine"
	getMachineStatusOperationLabel = "get_machine_status"
	getVolumeIDsOperationLabel     = "get_volume_ids"
)

// defaultDriver implements provider.Driver interface
type defaultDriver struct {
	factory access.Factory
}

// NewDefaultDriver creates a new instance of an implementation of provider.Driver. This can be mostly used by tests where we also wish to have our own polling intervals.
func NewDefaultDriver(accessFactory access.Factory) driver.Driver {
	return defaultDriver{
		factory: accessFactory,
	}
}

func (d defaultDriver) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (resp *driver.ListMachinesResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(listMachinesOperationLabel, &err)()
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return
	}
	vmNames, err := helpers.ExtractVMNamesFromVMsNICsDisks(ctx, d.factory, connectConfig, providerSpec.ResourceGroup, providerSpec)
	if err != nil {
		return
	}
	resp = helpers.ConstructMachineListResponse(providerSpec.Location, vmNames)
	return
}

func (d defaultDriver) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (resp *driver.CreateMachineResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(createMachineOperationLabel, &err)()

	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return
	}
	vmName := req.Machine.Name
	nicName := utils.CreateNICName(vmName)

	imageReference, plan, err := helpers.ProcessVMImageConfiguration(ctx, d.factory, connectConfig, providerSpec, vmName)
	if err != nil {
		return
	}
	subnet, err := helpers.GetSubnet(ctx, d.factory, connectConfig, providerSpec)
	if err != nil {
		return
	}

	nicID, err := helpers.CreateNICIfNotExists(ctx, d.factory, connectConfig, providerSpec, subnet, nicName)
	if err != nil {
		return
	}

	vm, err := helpers.CreateVM(ctx, d.factory, connectConfig, providerSpec, imageReference, plan, req.Secret, nicID, vmName)
	if err != nil {
		return
	}
	resp = helpers.ConstructCreateMachineResponse(providerSpec.Location, vmName)
	helpers.LogVMCreation(providerSpec.Location, providerSpec.ResourceGroup, vm)
	return
}

func (d defaultDriver) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (resp *driver.DeleteMachineResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(deleteMachineOperationLabel, &err)()

	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return
	}
	var (
		resourceGroup = providerSpec.ResourceGroup
		vmName        = strings.ToLower(req.Machine.Name)
	)
	// Check if Deletion of the machine (VM, NIC, Disks) can be completely skipped.
	skipDelete, err := helpers.SkipDeleteMachine(ctx, d.factory, connectConfig, resourceGroup)
	if err != nil {
		return
	}
	if skipDelete {
		klog.Warningf("skipping delete of Machine [ResourceGroup: %s, Name: %s] since the ResourceGroup no longer exists", resourceGroup, req.Machine.Name)
		resp = &driver.DeleteMachineResponse{}
		return
	}

	vmAccess, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		err = status.WrapError(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v\n", resourceGroup, vmName, err), err)
		return
	}
	vm, err := clienthelpers.GetVirtualMachine(ctx, vmAccess, resourceGroup, vmName)
	if err != nil {
		err = status.WrapError(codes.Internal, fmt.Sprintf("failed to get virtual machine for VM: [resourceGroup: %s, name: %s], Err: %v", resourceGroup, vmName, err), err)
		return
	}
	/*
		It is possible to have left over NIC's and Disks even if the VM is no longer there. This is made possible because in the earlier version of this provider
		implementation the cascade-delete is not enabled for NICs and Disks on deletion of the VM. Thus, it's possible that while the VM gets deleted the NIC's and Disks are left behind.
		Once all the VirtualMachines are launched with cascade-delete enabled for NICs and Disks then this can be removed.
	*/
	if vm == nil {
		klog.Infof("VirtualMachine [resourceGroup: %s, name: %s] does not exist. Skipping deletion of VirtualMachine. Checking for leftover NICs and Disks and if present delete tasks will be added.", providerSpec.ResourceGroup, vmName)
		// check if there are leftover NICs and Disks that needs to be deleted.
		if err = helpers.CheckAndDeleteLeftoverNICsAndDisks(ctx, d.factory, vmName, connectConfig, providerSpec); err != nil {
			return
		}
	} else {
		if helpers.CanUpdateVirtualMachine(vm) {
			if err = helpers.UpdateCascadeDeleteOptions(ctx, providerSpec, vmAccess, resourceGroup, vm); err != nil {
				return
			}
			if err = helpers.DeleteVirtualMachine(ctx, vmAccess, resourceGroup, vmName); err != nil {
				return
			}
		} else {
			klog.Infof("Cannot update VM: [ResourceGroup: %s, Name: %s]. Either the VM has provisionState set to Failed or there are one or more data disks that are marked for detachment, update call to this VM will fail and therefore skipped. Will now delete the VM and all its associated resources.", resourceGroup, vmName)
			if err = helpers.DeleteVirtualMachine(ctx, vmAccess, resourceGroup, vmName); err != nil {
				return
			}
			if err = helpers.CheckAndDeleteLeftoverNICsAndDisks(ctx, d.factory, vmName, connectConfig, providerSpec); err != nil {
				return
			}
		}
		klog.Infof("Successfully deleted all Machine resources[VM, NIC, Disks] for [ResourceGroup: %s, VMName: %s]", providerSpec.ResourceGroup, vmName)
	}
	resp = &driver.DeleteMachineResponse{}
	return
}

func (d defaultDriver) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (resp *driver.GetMachineStatusResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(getMachineStatusOperationLabel, &err)()

	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}

	resourceGroup := providerSpec.ResourceGroup
	vmName := req.Machine.Name
	vmAccess, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		err = status.WrapError(codes.Internal, fmt.Sprintf("Failed to create virtual machine access to process request: [ResourceGroup: %s, VMName: %s], Err: %v", resourceGroup, vmName, err), err)
		return
	}

	// TODO: After getting response for Query: [https://github.com/Azure/azure-sdk-for-go/issues/21031] replace this call with a more optimized variant to check if a VM exists.
	vm, err := clienthelpers.GetVirtualMachine(ctx, vmAccess, resourceGroup, vmName)
	if err != nil {
		err = status.WrapError(codes.Internal, fmt.Sprintf("Failed to get VM: [ResourceGroup: %s, Name: %s], Err: %v", resourceGroup, vmName, err), err)
		return
	}
	if vm == nil {
		err = status.Error(codes.NotFound, fmt.Sprintf("VM: [ResourceGroup: %s, Name: %s] is not found", resourceGroup, vmName))
		return
	}
	// TODO: Enhance the response as proposed in [https://github.com/gardener/machine-controller-manager-provider-azure/issues/88] once that is taken up.
	klog.Infof("VM found for [Machine: %s, ResourceGroup: %s]", vmName, resourceGroup)
	resp = helpers.ConstructGetMachineStatusResponse(providerSpec.Location, vmName)
	return
}

func (d defaultDriver) GetVolumeIDs(_ context.Context, request *driver.GetVolumeIDsRequest) (resp *driver.GetVolumeIDsResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(getVolumeIDsOperationLabel, &err)()

	var volumeIDs []string

	if request.PVSpecs != nil {
		for _, pvSpec := range request.PVSpecs {
			if pvSpec.AzureDisk != nil {
				volumeIDs = append(volumeIDs, pvSpec.AzureDisk.DiskName)
			} else if pvSpec.CSI != nil && pvSpec.CSI.Driver == utils.AzureCSIDriverName && !utils.IsEmptyString(pvSpec.CSI.VolumeHandle) {
				volumeIDs = append(volumeIDs, pvSpec.CSI.VolumeHandle)
			}
		}
	}
	resp = &driver.GetVolumeIDsResponse{VolumeIDs: volumeIDs}
	return
}
