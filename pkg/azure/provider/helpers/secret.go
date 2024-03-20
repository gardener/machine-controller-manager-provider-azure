// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"fmt"
	"strings"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api/validation"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
)

// ValidateSecretAndCreateConnectConfig validates the secret and creates an instance of azure.ConnectConfig out of it.
func ValidateSecretAndCreateConnectConfig(secret *corev1.Secret) (access.ConnectConfig, error) {
	if err := validation.ValidateProviderSecret(secret); err != nil {
		return access.ConnectConfig{}, status.Error(codes.InvalidArgument, fmt.Sprintf("error in validating secret: %v", err))
	}

	var (
		subscriptionID = ExtractCredentialsFromData(secret.Data, api.SubscriptionID, api.AzureSubscriptionID)
		tenantID       = ExtractCredentialsFromData(secret.Data, api.TenantID, api.AzureTenantID)
		clientID       = ExtractCredentialsFromData(secret.Data, api.ClientID, api.AzureClientID)
		clientSecret   = ExtractCredentialsFromData(secret.Data, api.ClientSecret, api.AzureClientSecret)
	)
	return access.ConnectConfig{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
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
