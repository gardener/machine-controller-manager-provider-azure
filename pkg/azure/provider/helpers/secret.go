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
