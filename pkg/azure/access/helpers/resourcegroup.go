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
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	resourceGroupExistsServiceLabel = "resource_group_exists"
)

// ResourceGroupExists checks if the given resourceGroup exists.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func ResourceGroupExists(ctx context.Context, client *armresources.ResourceGroupsClient, resourceGroup string) (exists bool, err error) {
	defer instrument.AZAPIMetricRecorderFn(resourceGroupExistsServiceLabel, &err)()

	resp, err := client.CheckExistence(ctx, resourceGroup, nil)
	if err != nil {
		if errors.IsNotFoundAzAPIError(err) {
			exists = resp.Success
			err = nil
			return
		}
		errors.LogAzAPIError(err, "Failed to check if ResourceGroup: %s exists", resourceGroup)
		return false, err
	}
	exists = resp.Success
	return
}
