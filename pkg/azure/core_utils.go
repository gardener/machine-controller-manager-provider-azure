/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package azure contains the cloud provider specific implementations to manage machines
package azure

import (
	"encoding/json"
	"fmt"

	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis/validation"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
)

// DecodeProviderSpecAndSecret unmarshals the raw providerspec into api.AzureProviderSpec structure
func DecodeProviderSpecAndSecret(machineClass *v1alpha1.MachineClass, secret *corev1.Secret) (*api.AzureProviderSpec, error) {
	var providerSpec *api.AzureProviderSpec

	// Extract providerSpec
	err := json.Unmarshal(machineClass.ProviderSpec.Raw, &providerSpec)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	//Validate the Spec and Secrets
	ValidationErr := validation.ValidateAzureSpecNSecret(providerSpec, secret)
	if ValidationErr != nil {
		err = fmt.Errorf("error while validating ProviderSpec %v", ValidationErr)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return providerSpec, nil
}

// encodeMachineID generates the providerID for the nodes
func encodeMachineID(location, vmName string) string {
	return fmt.Sprintf("azure:///%s/%s", location, vmName)
}
