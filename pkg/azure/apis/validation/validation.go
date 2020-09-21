/*
Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package validation - validation is used to validate cloud specific ProviderSpec
package validation

import (
	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	corev1 "k8s.io/api/core/v1"
)

// ValidateAzureSpecNSecret validates provider spec and secret to check if all fields are present and valid
func ValidateAzureSpecNSecret(spec *api.AzureProviderSpec, secrets *corev1.Secret) []error {
	// Code for validation of providerSpec goes here
	return nil
}
