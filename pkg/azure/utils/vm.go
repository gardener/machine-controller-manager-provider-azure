package utils

import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"

// DataDisksMarkedForDetachment checks if there is at least DataDisk that is marked for detachment.
// If there are no DataDisk(s) configured then it will return false.
func DataDisksMarkedForDetachment(vm *armcompute.VirtualMachine) bool {
	if vm.Properties != nil && !IsSliceNilOrEmpty(vm.Properties.StorageProfile.DataDisks) {
		for _, dataDisk := range vm.Properties.StorageProfile.DataDisks {
			if dataDisk.ToBeDetached != nil && *dataDisk.ToBeDetached {
				return true
			}
		}
	}
	return false
}
