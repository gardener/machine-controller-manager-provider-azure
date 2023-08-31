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

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	defer instrument.RecordDriverAPIMetric(err, listMachinesOperationLabel, time.Now())
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return
	}
	// azure resource graph uses KUSTO as their query language.
	// For additional information on KUSTO start here: [https://learn.microsoft.com/en-us/azure/data-explorer/kusto/query/]
	resGraphClient, err := d.factory.GetResourceGraphAccess(connectConfig)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("Failed to create resource graph access, Err: %v", err))
		return
	}
	vmNames, err := clienthelpers.ExtractVMNamesFromVirtualMachinesAndNICs(ctx, resGraphClient, connectConfig.SubscriptionID, providerSpec.ResourceGroup)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("Failed to extract VM names from VMs and NICs for resourceGroup: %s, Err: %v", providerSpec.ResourceGroup, err))
		return
	}
	resp = helpers.ConstructMachineListResponse(providerSpec.Location, vmNames)
	return
}

func (d defaultDriver) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (resp *driver.CreateMachineResponse, err error) {
	defer instrument.RecordDriverAPIMetric(err, createMachineOperationLabel, time.Now())
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

	vm, err := helpers.CreateOrUpdateVM(ctx, d.factory, connectConfig, providerSpec, imageReference, plan, req.Secret, nicID, vmName)
	if err != nil {
		return
	}
	resp = helpers.ConstructCreateMachineResponse(providerSpec.Location, vmName)
	helpers.LogVMCreation(providerSpec.Location, providerSpec.ResourceGroup, vm)
	return
}

func (d defaultDriver) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (resp *driver.DeleteMachineResponse, err error) {
	defer instrument.RecordDriverAPIMetric(err, deleteMachineOperationLabel, time.Now())
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
		klog.Infof("skipping delete of Machine [ResourceGroup: %s, Name: %s] since the ResourceGroup no longer exists", resourceGroup, req.Machine.Name)
		resp = &driver.DeleteMachineResponse{}
		return
	}

	vmAccess, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v\n", resourceGroup, vmName, err))
		return
	}
	vm, err := clienthelpers.GetVirtualMachine(ctx, vmAccess, resourceGroup, vmName)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("failed to get virtual machine for VM: [resourceGroup: %s, name: %s], Err: %v", resourceGroup, vmName, err))
		return
	}
	if vm == nil {
		klog.Infof("VirtualMachine [resourceGroup: %s, name: %s] does not exist. Skipping deletion of VirtualMachine", providerSpec.ResourceGroup, vmName)
		// check if there are leftover NICs and Disks that needs to be deleted.
		if err = helpers.CheckAndDeleteLeftoverNICsAndDisks(ctx, d.factory, vmName, connectConfig, providerSpec); err != nil {
			return
		}
	} else {
		err = helpers.UpdateCascadeDeleteOptionsAndDeleteVM(ctx, vmAccess, resourceGroup, vm)
		if err != nil {
			return
		}
		klog.Infof("Successfully delete all Machine resources[VM, NIC, Disks] for [ResourceGroup: %s, VMName: %s]", providerSpec.ResourceGroup, vmName)
	}
	resp = &driver.DeleteMachineResponse{}
	return
}

func (d defaultDriver) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (resp *driver.GetMachineStatusResponse, err error) {
	defer instrument.RecordDriverAPIMetric(err, getMachineStatusOperationLabel, time.Now())
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}

	resourceGroup := providerSpec.ResourceGroup
	vmName := req.Machine.Name
	vmAccess, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("Failed to create virtual machine access to process request: [ResourceGroup: %s, VMName: %s], Err: %v", resourceGroup, vmName, err))
		return
	}

	// After getting response for Query: [https://github.com/Azure/azure-sdk-for-go/issues/21031] replace this call with a more optimized variant to check if a VM exists.
	vm, err := clienthelpers.GetVirtualMachine(ctx, vmAccess, resourceGroup, vmName)
	if err != nil {
		err = status.Error(codes.Internal, fmt.Sprintf("Failed to get VM: [ResourceGroup: %s, Name: %s], Err: %v", resourceGroup, vmName, err))
		return
	}
	if vm == nil {
		err = status.Error(codes.NotFound, fmt.Sprintf("VM: [ResourceGroup: %s, Name: %s] is not found", resourceGroup, vmName))
		return
	}
	// Enhance the response as proposed in [https://github.com/gardener/machine-controller-manager-provider-azure/issues/88] once that is taken up.
	resp = helpers.ConstructGetMachineStatusResponse(providerSpec.Location, vmName)
	return
}

func (d defaultDriver) GetVolumeIDs(_ context.Context, request *driver.GetVolumeIDsRequest) (resp *driver.GetVolumeIDsResponse, err error) {
	defer instrument.RecordDriverAPIMetric(err, getVolumeIDsOperationLabel, time.Now())
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
