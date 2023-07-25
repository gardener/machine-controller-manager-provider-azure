package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	clienthelpers "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/provider/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
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

func (d defaultDriver) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}
	// azure resource graph uses KUSTO as their query language.
	// For additional information on KUSTO start here: [https://learn.microsoft.com/en-us/azure/data-explorer/kusto/query/]
	resGraphClient, err := d.factory.GetResourceGraphAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create resource graph access, Err: %v", err))
	}
	vmNames, err := clienthelpers.ExtractVMNamesFromVirtualMachinesAndNICs(ctx, resGraphClient, connectConfig.SubscriptionID, providerSpec.ResourceGroup)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to extract VM names from VMs and NICs for resourceGroup: %s, Err: %v", providerSpec.ResourceGroup, err))
	}
	return helpers.ConstructMachineListResponse(providerSpec.Location, vmNames), nil
}

func (d defaultDriver) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}
	vmName := req.Machine.Name
	nicName := utils.CreateNICName(vmName)

	imageReference, purchasePlan, err := helpers.ProcessVMImageConfiguration(ctx, d.factory, connectConfig, providerSpec, vmName)
	if err != nil {
		return nil, err
	}
	subnet, err := helpers.GetSubnet(ctx, d.factory, connectConfig, providerSpec)
	if err != nil {
		return nil, err
	}

	nicID, err := helpers.CreateNICIfNotExists(ctx, d.factory, connectConfig, providerSpec, subnet, nicName)
	if err != nil {
		return nil, err
	}

	helpers.CreateOrUpdateVM(ctx, d.factory, connectConfig, providerSpec, imageReference, purchasePlan, nicID, vmName)

	return helpers.ConstructCreateMachineResponse(providerSpec.Location, ""), nil
}

func (d defaultDriver) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}
	var (
		resourceGroup = providerSpec.ResourceGroup
		vmName        = strings.ToLower(req.Machine.Name)
	)
	// Check if Deletion of the machine (VM, NIC, Disks) can be completely skipped.
	skipDelete, err := helpers.SkipDeleteMachine(ctx, d.factory, connectConfig, resourceGroup)
	if err != nil {
		return nil, err
	}
	if skipDelete {
		klog.Infof("skipping delete of Machine [ResourceGroup: %s, Name: %s] since the ResourceGroup no longer exists", resourceGroup, req.Machine.Name)
		return &driver.DeleteMachineResponse{}, nil
	}

	vmAccess, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v\n", resourceGroup, vmName, err))
	}
	vm, err := clienthelpers.GetVirtualMachine(ctx, vmAccess, resourceGroup, vmName)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get virtual machine for VM: [resourceGroup: %s, name: %s], Err: %v", resourceGroup, vmName, err))
	}
	if vm == nil {
		klog.Infof("VirtualMachine [resourceGroup: %s, name: %s] does not exist. Skipping deletion of VirtualMachine", providerSpec.ResourceGroup, vmName)
		// check if there are leftover NICs and Disks that needs to be deleted.
		if err = helpers.CheckAndDeleteLeftoverNICsAndDisks(ctx, d.factory, vmName, connectConfig, providerSpec); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to check if there are left over resources for non-existent VM: [resourceGroup: %s, name: %s], Err: %v\n", resourceGroup, vmName, err))
		}
	} else {
		// update the VM and set cascade delete on NIC and Disks (OSDisk and DataDisks) if not already set and then trigger VM deletion.
		err = clienthelpers.SetCascadeDeleteForNICsAndDisks(ctx, vmAccess, resourceGroup, vm)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update cascade delete of associated resources for VM: [resourceGroup: %s, name: %s], Err: %v\\n", resourceGroup, vmName, err))
		}
		err = clienthelpers.DeleteVirtualMachine(ctx, vmAccess, resourceGroup, vmName)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete VM: [resourceGroup: %s, name: %s], Err: %v\\n", resourceGroup, vmName, err))
		}
	}

	return &driver.DeleteMachineResponse{}, nil
}

func (d defaultDriver) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}

	resourceGroup := providerSpec.ResourceGroup
	vmName := req.Machine.Name
	vmAccess, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v", resourceGroup, vmName, err))
	}

	// After getting response for Query: [https://github.com/Azure/azure-sdk-for-go/issues/21031] replace this call with a more optimized variant to check if a VM exists.
	vm, err := clienthelpers.GetVirtualMachine(ctx, vmAccess, resourceGroup, vmName)
	if err != nil {
		return nil, err
	}
	if vm == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("virtual machine [ResourceGroup: %s, Name: %s] is not found", resourceGroup, vmName))
	}

	// Enhance the response as proposed in [https://github.com/gardener/machine-controller-manager-provider-azure/issues/88] once that is taken up.
	return helpers.ConstructGetMachineStatusResponse(providerSpec.Location, vmName), nil
}

func (d defaultDriver) GetVolumeIDs(_ context.Context, request *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
	const csiDriverName = "disk.csi.azure.com"
	var volumeIDs []string

	if request.PVSpecs != nil {
		for _, pvSpec := range request.PVSpecs {
			if pvSpec.AzureDisk != nil {
				volumeIDs = append(volumeIDs, pvSpec.AzureDisk.DiskName)
			} else if pvSpec.CSI != nil && pvSpec.CSI.Driver == csiDriverName && !utils.IsEmptyString(pvSpec.CSI.VolumeHandle) {
				volumeIDs = append(volumeIDs, pvSpec.CSI.VolumeHandle)
			}
		}
	}

	return &driver.GetVolumeIDsResponse{VolumeIDs: volumeIDs}, nil
}
