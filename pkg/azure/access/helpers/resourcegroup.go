// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
