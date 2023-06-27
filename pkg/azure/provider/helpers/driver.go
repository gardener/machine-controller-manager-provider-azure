package helpers

import (
	"fmt"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/validation"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	corev1 "k8s.io/api/core/v1"
)

func ExtractProviderSpecAndConnectConfig(mcc *v1alpha1.MachineClass, secret *corev1.Secret) (api.AzureProviderSpec, client.ConnectConfig, error) {
	var (
		err           error
		providerSpec  api.AzureProviderSpec
		connectConfig client.ConnectConfig
	)
	// validate provider Spec provider. Exit early if it is not azure.
	if err = validation.ValidateMachineClassProvider(mcc); err != nil {
		return providerSpec, connectConfig, err
	}
	// unmarshall raw provider Spec from MachineClass and validate it. If validation fails return an error else return decoded spec.
	if providerSpec, err = DecodeAndValidateMachineClassProviderSpec(mcc); err != nil {
		return api.AzureProviderSpec{}, client.ConnectConfig{}, err
	}
	// validate secret and extract connect config required to create clients.
	if connectConfig, err = helpers.ValidateSecretAndCreateConnectConfig(secret); err != nil {
		return api.AzureProviderSpec{}, client.ConnectConfig{}, err
	}
	return providerSpec, connectConfig, nil
}

func CreateMachineListResponse(location string, vmNames []string) *driver.ListMachinesResponse {
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

func CreateMachineStatusResponse(location string, vmName string) *driver.GetMachineStatusResponse {
	instanceID := DeriveInstanceID(location, vmName)
	return &driver.GetMachineStatusResponse{
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
