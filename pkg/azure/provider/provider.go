package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client"
	clienthelpers "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/provider/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"
)

// driverProvider implements provider.Driver interface
type driverProvider struct {
	clientProvider client.ARMClientProvider
}

// NewDriver creates a new instance of an implementation of provider.Driver. This can be mostly used by tests where we also wish to have our own polling intervals.
func NewDriver(clientProvider client.ARMClientProvider) driver.Driver {
	return driverProvider{
		clientProvider: clientProvider,
	}
}

func (d driverProvider) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}
	// azure resource graph uses KUSTO as their query language.
	// For additional information on KUSTO start here: [https://learn.microsoft.com/en-us/azure/data-explorer/kusto/query/]
	resGraphClient, err := d.clientProvider.CreateResourceGraphClient(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create resource graph client, Err: %v", err))
	}
	vmNames, err := clienthelpers.ExtractVMNamesFromVirtualMachinesAndNICs(ctx, resGraphClient, connectConfig.SubscriptionID, providerSpec.ResourceGroup)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to extract VM names from VMs and NICs for resourceGroup: %s, Err: %v", providerSpec.ResourceGroup, err))
	}
	return helpers.CreateMachineListResponse(providerSpec.Location, vmNames)
}

func (d driverProvider) CreateMachine(ctx context.Context, request *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d driverProvider) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
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

	vmClient, err := d.clientProvider.CreateVirtualMachinesClient(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine client to process request: [resourceGroup: %s, vmName: %s], Err: %v", resourceGroup, vmName, err))
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
		// update the VM and set cascade delete on NIC and Disks (OSDisk and DataDisks)	and the trigger VM deletion.
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

// skipDeleteMachine checks if ResourceGroup exists. If it does not exist then there is no need to delete any resource as it is assumed that none would exist.
func (d driverProvider) skipDeleteMachine(ctx context.Context, connectConfig client.ConnectConfig, resourceGroup string) (bool, error) {
	resGroupCli, err := d.clientProvider.CreateResourceGroupsClient(connectConfig)
	if err != nil {
		return false, status.Error(codes.Internal, fmt.Sprintf("failed to create resource group client to process request: [resourceGroup: %s]"))
	}
	resGroupExists, err := clienthelpers.ResourceGroupExists(ctx, resGroupCli, resourceGroup)
	if err != nil {
		return false, status.Error(codes.Internal, fmt.Sprintf("failed to check if resource group %s exists, Err: %v", resourceGroup, err))
	}
	return !resGroupExists, nil
}

func (d driverProvider) getVirtualMachine(ctx context.Context, connectConfig client.ConnectConfig, resourceGroup, vmName string) (*armcompute.VirtualMachine, error) {
	vmClient, err := d.clientProvider.CreateVirtualMachinesClient(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine client to process request: [resourceGroup: %s, vmName: %s], Err: %v", resourceGroup, vmName, err))
	}
	return clienthelpers.GetVirtualMachine(ctx, vmClient, resourceGroup, vmName)
}

func (d driverProvider) checkAndDeleteLeftoverNICsAndDisks(ctx context.Context, vmName string, connectConfig client.ConnectConfig, providerSpec api.AzureProviderSpec) error {
	// Gather the names for NIC, OSDisk and Data Disks that needs to be checked for existence and then deleted if they exist.
	resourceGroup := providerSpec.ResourceGroup
	nicName := helpers.CreateNICName(vmName)
	diskNames := helpers.GetDiskNames(providerSpec, vmName)

	// create NIC and Disks clients
	nicClient, err := d.clientProvider.CreateNetworkInterfacesClient(connectConfig)
	if err != nil {
		return err
	}
	disksClient, err := d.clientProvider.CreateDisksClient(connectConfig)
	if err != nil {
		return err
	}

	// Create NIC and Disk deletion tasks and run them concurrently.
	tasks := make([]utils.Task, 0, len(diskNames)+1)
	tasks = append(tasks, d.createNICDeleteTask(resourceGroup, nicName, nicClient))
	tasks = append(tasks, d.createDiskDeletionTasks(resourceGroup, diskNames, disksClient)...)
	return errors.Join(utils.RunConcurrently(ctx, tasks, len(tasks))...)
}

func (d driverProvider) createNICDeleteTask(resourceGroup, nicName string, nicClient *armnetwork.InterfacesClient) utils.Task {
	return utils.Task{
		Name: fmt.Sprintf("delete-nic-[resourceGroup: %s name: %s]", resourceGroup, nicName),
		Fn: func(ctx context.Context) error {
			return clienthelpers.DeleteNICIfExists(ctx, nicClient, resourceGroup, nicName)
		},
	}
}

func (d driverProvider) createDiskDeletionTasks(resourceGroup string, diskNames []string, diskClient *armcompute.DisksClient) []utils.Task {
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

func (d driverProvider) GetMachineStatus(ctx context.Context, request *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d driverProvider) GetVolumeIDs(_ context.Context, request *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
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
