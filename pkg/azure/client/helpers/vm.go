package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	vmGETServiceLabel = "virtual-machine-get"
)

func GetVirtualMachine(ctx context.Context, client *armcompute.VirtualMachinesClient, resourceGroup, vmName string) (vm *armcompute.VirtualMachine, exists bool, err error) {
	var getResp armcompute.VirtualMachinesClientGetResponse
	defer instrument.RecordAzAPIMetric(err, vmGETServiceLabel, time.Now())
	getResp, err = client.Get(ctx, resourceGroup, vmName, nil)
	if err != nil {
		if errors.IsNotFoundAzAPIError(err) {
			return nil, false, nil
		}
		return
	}
	vm = &getResp.VirtualMachine
	return
}

func IsVMCascadeDeleteSetForNICs(vm *armcompute.VirtualMachine) bool {
	var result bool
	if vm != nil && vm.Properties != nil {
		nwProfile := vm.Properties.NetworkProfile
		if nwProfile != nil && len(nwProfile.NetworkInterfaces) > 0 {
			result = true // this is set to true here so that we can apply a conjunction
			for _, nicRef := range nwProfile.NetworkInterfaces {
				var cascadeDeleteSet bool
				if nicRef.Properties != nil {
					deleteOption := nicRef.Properties.DeleteOption
					cascadeDeleteSet = deleteOption != nil && *deleteOption == armcompute.DeleteOptionsDelete
				}
				result = result && cascadeDeleteSet
			}
		}
	}
	return result
}

func IsVMCascadeDeleteSetForOSDisks(vm *armcompute.VirtualMachine) bool {
	if vm != nil && vm.Properties != nil {
		storageProfile := vm.Properties.StorageProfile
		if storageProfile != nil && storageProfile.OSDisk != nil {
			deleteOption := storageProfile.OSDisk.DeleteOption
			return deleteOption != nil && *deleteOption == armcompute.DiskDeleteOptionTypesDelete
		}
	}
	return false
}

func IsVMCascadeDeleteSetForDataDisks(vm *armcompute.VirtualMachine) bool {
	var result bool
	if vm != nil && vm.Properties != nil {
		storageProfile := vm.Properties.StorageProfile
		if storageProfile != nil && storageProfile.DataDisks != nil && len(storageProfile.DataDisks) > 0 {
			result = true
			var cascadeDeleteSet bool
			for _, disk := range storageProfile.DataDisks {
				cascadeDeleteSet = disk != nil && disk.DeleteOption != nil && *disk.DeleteOption == armcompute.DiskDeleteOptionTypesDelete
			}
			result = result && cascadeDeleteSet
		}
	}
	return result
}
