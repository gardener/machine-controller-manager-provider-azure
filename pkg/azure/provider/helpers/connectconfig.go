// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"strings"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api/validation"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
)

// ValidateSecretAndCreateConnectConfig validates the secret and creates an instance of azure.ConnectConfig out of it.
func ValidateSecretAndCreateConnectConfig(secret *corev1.Secret, cloudConfiguration *api.CloudConfiguration) (access.ConnectConfig, error) {
	if err := validation.ValidateProviderSecret(secret); err != nil {
		return access.ConnectConfig{}, status.Error(codes.InvalidArgument, fmt.Sprintf("error in validating secret: %v", err))
	}

	var (
		subscriptionID       = ExtractCredentialsFromData(secret.Data, api.SubscriptionID, api.AzureSubscriptionID)
		tenantID             = ExtractCredentialsFromData(secret.Data, api.TenantID, api.AzureTenantID)
		clientID             = ExtractCredentialsFromData(secret.Data, api.ClientID, api.AzureClientID)
		clientSecret         = ExtractCredentialsFromData(secret.Data, api.ClientSecret, api.AzureClientSecret)
		azCloudConfiguration = DetermineAzureCloudConfiguration(cloudConfiguration)
	)

	return access.ConnectConfig{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		ClientOptions:  azcore.ClientOptions{Cloud: azCloudConfiguration},
	}, nil
}

// ExtractCredentialsFromData extracts and trims a value from the given data map. The first key that exists is being
// returned, otherwise, the next key is tried, etc. If no key exists then an empty string is returned.
func ExtractCredentialsFromData(data map[string][]byte, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			return strings.TrimSpace(string(val))
		}
	}
	return ""
}

// DetermineAzureCloudConfiguration returns the Azure cloud.Configuration corresponding to the instance given by the provided api.Configuration.
func DetermineAzureCloudConfiguration(cloudConfiguration *api.CloudConfiguration) cloud.Configuration {
	if cloudConfiguration != nil {
		cloudConfigurationName := cloudConfiguration.Name
		switch {
		case strings.EqualFold(cloudConfigurationName, api.CloudNamePublic):
			return cloud.AzurePublic
		case strings.EqualFold(cloudConfigurationName, api.CloudNameGov):
			return cloud.AzureGovernment
		case strings.EqualFold(cloudConfigurationName, api.CloudNameChina):
			return cloud.AzureChina
		default:
			return cloud.AzurePublic
		}
	}
	// Fallback
	return cloud.AzurePublic
}