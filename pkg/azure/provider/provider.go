package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	clienthelpers "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
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

	_, err = d.createNICIfNotExists(ctx, providerSpec, connectConfig, vmName)
	if err != nil {
		return nil, err
	}

	d.createOrUpdateVM(ctx, connectConfig, providerSpec, vmName)

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
	skipDelete, err := d.skipDeleteMachine(ctx, connectConfig, resourceGroup)
	if err != nil {
		return nil, err
	}
	if skipDelete {
		klog.Infof("skipping delete of Machine [ResourceGroup: %s, Name: %s] since the ResourceGroup no longer exists", resourceGroup, req.Machine.Name)
		return &driver.DeleteMachineResponse{}, nil
	}

	vmClient, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v", resourceGroup, vmName, err))
	}
	vm, err := clienthelpers.GetVirtualMachine(ctx, vmClient, resourceGroup, vmName)
	if err != nil {
		return nil, err
	}
	if vm == nil {
		klog.Infof("VirtualMachine [resourceGroup: %s, name: %s] does not exist. Skipping deletion of VirtualMachine", providerSpec.ResourceGroup, vmName)
		// check if there are leftover NICs and Disks that needs to be deleted.
		if err = d.checkAndDeleteLeftoverNICsAndDisks(ctx, vmName, connectConfig, providerSpec); err != nil {
			return nil, err
		}
	} else {
		// update the VM and set cascade delete on NIC and Disks (OSDisk and DataDisks) if not already set and then trigger VM deletion.
		err = clienthelpers.SetCascadeDeleteForNICsAndDisks(ctx, vmClient, resourceGroup, vm)
		if err != nil {
			return nil, err
		}
		err = clienthelpers.DeleteVirtualMachine(ctx, vmClient, resourceGroup, vmName)
		if err != nil {
			return nil, err
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
	vmClient, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v", resourceGroup, vmName, err))
	}

	// After getting response for Query: [https://github.com/Azure/azure-sdk-for-go/issues/21031] replace this call with a more optimized variant to check if a VM exists.
	vm, err := clienthelpers.GetVirtualMachine(ctx, vmClient, resourceGroup, vmName)
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

// skipDeleteMachine checks if ResourceGroup exists. If it does not exist then there is no need to delete any resource as it is assumed that none would exist.
func (d defaultDriver) skipDeleteMachine(ctx context.Context, connectConfig access.ConnectConfig, resourceGroup string) (bool, error) {
	resGroupCli, err := d.factory.GetResourceGroupsAccess(connectConfig)
	if err != nil {
		return false, status.Error(codes.Internal, fmt.Sprintf("failed to create resource group access to process request: [resourceGroup: %s]", resourceGroup))
	}
	resGroupExists, err := clienthelpers.ResourceGroupExists(ctx, resGroupCli, resourceGroup)
	if err != nil {
		return false, status.Error(codes.Internal, fmt.Sprintf("failed to check if resource group %s exists, Err: %v", resourceGroup, err))
	}
	return !resGroupExists, nil
}

func (d defaultDriver) getVirtualMachine(ctx context.Context, connectConfig access.ConnectConfig, resourceGroup, vmName string) (*armcompute.VirtualMachine, error) {
	vmClient, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v", resourceGroup, vmName, err))
	}
	return clienthelpers.GetVirtualMachine(ctx, vmClient, resourceGroup, vmName)
}

func (d defaultDriver) checkAndDeleteLeftoverNICsAndDisks(ctx context.Context, vmName string, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec) error {
	// Gather the names for NIC, OSDisk and Data Disks that needs to be checked for existence and then deleted if they exist.
	resourceGroup := providerSpec.ResourceGroup
	nicName := helpers.CreateNICName(vmName)
	diskNames := helpers.GetDiskNames(providerSpec, vmName)

	// create NIC and Disks clients
	nicClient, err := d.factory.GetNetworkInterfacesAccess(connectConfig)
	if err != nil {
		return err
	}
	disksClient, err := d.factory.GetDisksAccess(connectConfig)
	if err != nil {
		return err
	}

	// Create NIC and Disk deletion tasks and run them concurrently.
	tasks := make([]utils.Task, 0, len(diskNames)+1)
	tasks = append(tasks, d.createNICDeleteTask(resourceGroup, nicName, nicClient))
	tasks = append(tasks, d.createDiskDeletionTasks(resourceGroup, diskNames, disksClient)...)
	return errors.Join(utils.RunConcurrently(ctx, tasks, len(tasks))...)
}

func (d defaultDriver) createNICDeleteTask(resourceGroup, nicName string, nicClient *armnetwork.InterfacesClient) utils.Task {
	return utils.Task{
		Name: fmt.Sprintf("delete-nic-[resourceGroup: %s name: %s]", resourceGroup, nicName),
		Fn: func(ctx context.Context) error {
			return clienthelpers.DeleteNICIfExists(ctx, nicClient, resourceGroup, nicName)
		},
	}
}

func (d defaultDriver) createDiskDeletionTasks(resourceGroup string, diskNames []string, diskClient *armcompute.DisksClient) []utils.Task {
	tasks := make([]utils.Task, 0, len(diskNames))
	for _, diskName := range diskNames {
		task := utils.Task{
			Name: fmt.Sprintf("delete-disk-[resourceGroup: %s name: %s]", resourceGroup, diskName),
			Fn: func(ctx context.Context) error {
				return clienthelpers.DeleteDiskIfExists(ctx, diskClient, resourceGroup, diskName)
			},
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func (d defaultDriver) createNICIfNotExists(ctx context.Context, providerSpec api.AzureProviderSpec, connectConfig access.ConnectConfig, vmName string) (string, error) {
	nicAccess, err := d.factory.GetNetworkInterfacesAccess(connectConfig)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("failed to create nic access, Err: %v", err))
	}
	subnetAccess, err := d.factory.GetSubnetAccess(connectConfig)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("failed to create subnet access, Err: %v", err))
	}
	return clienthelpers.CreateNICIfNotExists(ctx, nicAccess, subnetAccess, providerSpec, helpers.CreateNICName(vmName))
}

func (d defaultDriver) createOrUpdateVM(ctx context.Context, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec, vmName string) error {
	_, err := d.factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v", providerSpec.ResourceGroup, vmName, err))
	}
	// TODO
	return nil
}

func (d defaultDriver) getVirtualMachineImage(ctx context.Context, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec) (*armcompute.VirtualMachineImage, error) {
	vmImagesAccess, err := d.factory.GetVirtualMachineImagesAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create image access, Err: %v", err))
	}
	imgRef := helpers.GetImageReference(providerSpec)
	vmImage, err := clienthelpers.GetVMImage(ctx, vmImagesAccess, providerSpec.Location, imgRef)
	if err != nil {
		return nil, err
	}
	return vmImage, nil
}
