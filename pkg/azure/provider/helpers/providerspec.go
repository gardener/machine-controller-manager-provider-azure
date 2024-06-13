// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
)

// DetermineCloudConfiguration returns the Azure cloud.Configuration corresponding to the instance given by the provided api.Configuration.
func DetermineCloudConfiguration(cloudConfiguration *api.CloudConfiguration) (cloud.Configuration, error) {
	if cloudConfiguration != nil {
		cloudConfigurationName := cloudConfiguration.Name
		switch {
		case strings.EqualFold(cloudConfigurationName, api.CloudNamePublic):
			return cloud.AzurePublic, nil
		case strings.EqualFold(cloudConfigurationName, api.CloudNameGov):
			return cloud.AzureGovernment, nil
		case strings.EqualFold(cloudConfigurationName, api.CloudNameChina):
			return cloud.AzureChina, nil

		default:
			return cloud.Configuration{}, fmt.Errorf("unknown cloud configuration name '%s'", cloudConfigurationName)
		}
	} else {
		// Fallback
		return cloud.AzurePublic, nil
	}
}
