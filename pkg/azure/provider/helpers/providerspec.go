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

// ExtractCloudConfiguration retrieves the name of the cloud instance to connect to from the AzureProviderSpec
// and returns the corresponding azure cloud Configuration.
func ExtractCloudConfiguration(spec *api.AzureProviderSpec) (cloud.Configuration, error) {
	if spec.CloudConfiguration == nil {
		return cloud.AzurePublic, nil
	}

	cloudConfigurationName := spec.CloudConfiguration.Name
	switch {
	case strings.EqualFold(cloudConfigurationName, api.AzurePublicCloudName):
		return cloud.AzurePublic, nil
	case strings.EqualFold(cloudConfigurationName, api.AzureGovCloudName):
		return cloud.AzureGovernment, nil
	case strings.EqualFold(cloudConfigurationName, api.AzureChinaCloudName):
		return cloud.AzureChina, nil

	default:
		return cloud.Configuration{}, fmt.Errorf("unknown cloud configuration name '%s'", cloudConfigurationName)
	}
}
