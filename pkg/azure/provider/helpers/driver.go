package helpers

import (
	"fmt"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/validation"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	corev1 "k8s.io/api/core/v1"
)

func ExtractProviderSpecAndConnectConfig(mcc *v1alpha1.MachineClass, secret *corev1.Secret) (api.AzureProviderSpec, access.ConnectConfig, error) {
	var (
		err           error
		providerSpec  api.AzureProviderSpec
		connectConfig access.ConnectConfig
	)
	// validate provider Spec provider. Exit early if it is not azure.
	if err = validation.ValidateMachineClassProvider(mcc); err != nil {
		return providerSpec, connectConfig, err
	}
	// unmarshall raw provider Spec from MachineClass and validate it. If validation fails return an error else return decoded spec.
	if providerSpec, err = DecodeAndValidateMachineClassProviderSpec(mcc); err != nil {
		return api.AzureProviderSpec{}, access.ConnectConfig{}, err
	}
	// validate secret and extract connect config required to create clients.
	if connectConfig, err = helpers.ValidateSecretAndCreateConnectConfig(secret); err != nil {
		return api.AzureProviderSpec{}, access.ConnectConfig{}, err
	}
	return providerSpec, connectConfig, nil
}

func ConstructMachineListResponse(location string, vmNames []string) *driver.ListMachinesResponse {
	listMachineRes := driver.ListMachinesResponse{}
	instanceIdToVMNameMap := make(map[string]string, len(vmNames))
	if len(vmNames) == 0 {
		return &listMachineRes
	}
	for _, vmName := range vmNames {
		instanceIdToVMNameMap[DeriveInstanceID(location, vmName)] = vmName
	}
	listMachineRes.MachineList = instanceIdToVMNameMap
	return &listMachineRes
}

func ConstructGetMachineStatusResponse(location string, vmName string) *driver.GetMachineStatusResponse {
	instanceID := DeriveInstanceID(location, vmName)
	return &driver.GetMachineStatusResponse{
		ProviderID: instanceID,
		NodeName:   vmName,
	}
}

func ConstructCreateMachineResponse(location string, vmName string) *driver.CreateMachineResponse {
	instanceID := DeriveInstanceID(location, vmName)
	return &driver.CreateMachineResponse{
		ProviderID: instanceID,
		NodeName:   vmName,
	}
}

func DeriveInstanceID(location, vmName string) string {
	return fmt.Sprintf("azure:///%s/%s", location, vmName)
}

func GetDiskNames(providerSpec api.AzureProviderSpec, vmName string) []string {
	dataDisks := providerSpec.Properties.StorageProfile.DataDisks
	diskNames := make([]string, 0, len(dataDisks)+1)
	diskNames = append(diskNames, CreateOSDiskName(vmName))
	if !utils.IsSliceNilOrEmpty(dataDisks) {
		for _, disk := range dataDisks {
			diskName := CreateDataDiskName(vmName, disk)
			diskNames = append(diskNames, diskName)
		}
	}
	return diskNames
}
