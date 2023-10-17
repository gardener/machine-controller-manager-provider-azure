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

package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/validation"
)

// DecodeAndValidateMachineClassProviderSpec decodes v1alpha1.MachineClass.ProviderSpec.Raw into api.AzureProviderSpec.
// It also handles deprecated fields and ensures that the replacement fields are populated. A validated api.AzureProviderSpec
// is returned. In case there is an error during unmarshalling or validation an error will be returned.
func DecodeAndValidateMachineClassProviderSpec(mcc *v1alpha1.MachineClass) (api.AzureProviderSpec, error) {
	var providerSpec api.AzureProviderSpec
	// Extract providerSpec
	if err := json.Unmarshal(mcc.ProviderSpec.Raw, &providerSpec); err != nil {
		return api.AzureProviderSpec{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// api.AzureVirtualMachineProperties.MachineSet has been marked as deprecated.
	// If AzureProviderSpec still has MachineSet populated then also copy equivalent values
	// to the VirtualMachineScaleSet and AvailabilitySet. We do the validation for fields in MachineSet
	// here separately so that we can use the validated values to populate VirtualMachineScaleSet/AvailabilitySet.
	// TODO: This complete `if` condition should be removed once consumers no longer use MachineSetConfig.
	if providerSpec.Properties.MachineSet != nil {
		if err := validation.ValidateMachineSetConfig(providerSpec.Properties.MachineSet); err != nil {
			return api.AzureProviderSpec{}, status.Error(codes.InvalidArgument, fmt.Sprintf("error while validation providerSpec.Properties.MachineSet: %v", err))
		}
		if providerSpec.Properties.VirtualMachineScaleSet == nil && providerSpec.Properties.MachineSet.Kind == api.MachineSetKindVMO {
			providerSpec.Properties.VirtualMachineScaleSet = &api.AzureSubResource{ID: providerSpec.Properties.MachineSet.ID}
		}
		if providerSpec.Properties.AvailabilitySet == nil && providerSpec.Properties.MachineSet.Kind == api.MachineSetKindAvailabilitySet {
			providerSpec.Properties.AvailabilitySet = &api.AzureSubResource{ID: providerSpec.Properties.MachineSet.ID}
		}
	}

	if err := validation.ValidateProviderSpec(providerSpec); err != nil {
		return api.AzureProviderSpec{}, status.Error(codes.InvalidArgument, fmt.Sprintf("error in validation of AzureProviderSpec: %v", err))
	}

	return providerSpec, nil
}
