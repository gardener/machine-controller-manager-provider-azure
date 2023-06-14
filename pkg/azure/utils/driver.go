package utils

import (
	"fmt"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/types"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	corev1 "k8s.io/api/core/v1"
)

func ExtractProviderSpecAndConnectConfig(mcc *v1alpha1.MachineClass, secret *corev1.Secret) (*api.AzureProviderSpec, *types.ConnectConfig, error) {
	var (
		err           error
		providerSpec  *api.AzureProviderSpec
		connectConfig *types.ConnectConfig
	)
	// validate provider Spec provider. Exit early if it is not azure.
	if err = ValidateMachineClassProvider(mcc); err != nil {
		return nil, nil, err
	}
	// unmarshall raw provider Spec from MachineClass and validate it. If validation fails return an error else return decoded spec.
	if providerSpec, err = DecodeAndValidateMachineClassProviderSpec(mcc); err != nil {
		return nil, nil, err
	}
	// validate secret and extract connect config required to create clients.
	if connectConfig, err = ValidateSecretAndCreateConnectConfig(secret); err != nil {
		return nil, nil, err
	}
	return providerSpec, connectConfig, nil
}

func CreateMachineListResponse(location string, vmNames []string) (*driver.ListMachinesResponse, error) {
	listMachineRes := driver.ListMachinesResponse{}
	instanceIdToVMNameMap := make(map[string]string, len(vmNames))
	if len(vmNames) == 0 {
		return &listMachineRes, nil
	}
	for _, vmName := range vmNames {
		instanceIdToVMNameMap[DeriveInstanceID(location, vmName)] = vmName
	}
	listMachineRes.MachineList = instanceIdToVMNameMap
	return &listMachineRes, nil
}

func DeriveInstanceID(location, vmName string) string {
	return fmt.Sprintf("azure:///%s/%s", location, vmName)
}
