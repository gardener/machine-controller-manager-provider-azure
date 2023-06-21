package helpers

import (
	"fmt"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

const (
	nicSuffix      = "-nic"
	osDiskSuffix   = "-os-disk"
	dataDiskSuffix = "-data-disk"
)

func CreateNICName(vmName string) string {
	return fmt.Sprintf("%s-%s", vmName, nicSuffix)
}

func CreateOSDiskName(vmName string) string {
	return fmt.Sprintf("%s-%s", vmName, osDiskSuffix)
}

func CreateDataDiskName(vmName string, dataDisk api.AzureDataDisk) string {
	prefix := vmName
	infix := getDataDiskInfix(dataDisk)
	return fmt.Sprintf("%s-%s%s", prefix, infix, dataDiskSuffix)
}

func getDataDiskInfix(dataDisk api.AzureDataDisk) string {
	name := dataDisk.Name
	if utils.IsEmptyString(name) {
		return fmt.Sprintf("%d", dataDisk.Lun)
	}
	return fmt.Sprintf("%s-%d", name, dataDisk.Lun)
}
