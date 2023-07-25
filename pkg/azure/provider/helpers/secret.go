package helpers

import (
	"fmt"
	"strings"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/validation"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
)

// ValidateSecretAndCreateConnectConfig validates the secret and creates an instance of azure.ConnectConfig out of it.
func ValidateSecretAndCreateConnectConfig(secret *corev1.Secret) (access.ConnectConfig, error) {
	if err := validation.ValidateProviderSecret(secret); err != nil {
		return access.ConnectConfig{}, status.Error(codes.InvalidArgument, fmt.Sprintf("error in validating secret: %v", err))
	}

	var (
		subscriptionID = extractCredentialsFromData(secret.Data, api.SubscriptionID, api.AzureSubscriptionID)
		tenantID       = extractCredentialsFromData(secret.Data, api.TenantID, api.AzureTenantID)
		clientID       = extractCredentialsFromData(secret.Data, api.ClientID, api.AzureClientID)
		clientSecret   = extractCredentialsFromData(secret.Data, api.ClientSecret, api.AzureClientSecret)
	)
	return access.ConnectConfig{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}, nil
}

// extractCredentialsFromData extracts and trims a value from the given data map. The first key that exists is being
// returned, otherwise, the next key is tried, etc. If no key exists then an empty string is returned.
func extractCredentialsFromData(data map[string][]byte, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			return strings.TrimSpace(string(val))
		}
	}
	return ""
}
