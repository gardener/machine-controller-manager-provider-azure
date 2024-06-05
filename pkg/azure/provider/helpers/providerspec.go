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

// DetermineCloudConfiguration returns the Azure cloud.Configuration corresponding to the instance given by the provided input. If both cloudConfiguration and
// region are provided, cloudConfiguration takes precedence.
func DetermineCloudConfiguration(cloudConfiguration *api.CloudConfiguration, region *string) (cloud.Configuration, error) {

	if cloudConfiguration != nil {
		cloudConfigurationName := cloudConfiguration.Name
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
	} else if region != nil {
		return cloudConfigurationFromRegion(*region), nil
	} else {
		// Fallback, this case should only occur during testing as we expect the region to always be given in an actual live scenario.
		return cloud.AzurePublic, nil
	}
}

// cloudConfigurationFromRegion returns a matching cloudConfiguration corresponding to a well known cloud instance for the given region
func cloudConfigurationFromRegion(region string) cloud.Configuration {
	switch {
	case hasAnyPrefix(region, api.AzureGovRegionPrefixes...):
		return cloud.AzureGovernment
	case hasAnyPrefix(region, api.AzureChinaRegionPrefixes...):
		return cloud.AzureChina
	default:
		return cloud.AzurePublic
	}
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	lString := strings.ToLower(s)
	for _, p := range prefixes {
		if strings.HasPrefix(lString, strings.ToLower(p)) {
			return true
		}
	}
	return false
}
