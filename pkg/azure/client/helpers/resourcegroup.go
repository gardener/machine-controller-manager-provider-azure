package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	resourceGroupExistsServiceLabel = "resource_group_exists"
)

// ResourceGroupExists checks if the given resourceGroup exists.
func ResourceGroupExists(ctx context.Context, client *armresources.ResourceGroupsClient, resourceGroup string) (exists bool, err error) {
	defer instrument.RecordAzAPIMetric(err, resourceGroupExistsServiceLabel, time.Now())
	resp, err := client.CheckExistence(ctx, resourceGroup, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to check if ResourceGroup: %s exists", resourceGroup)
		return false, err
	}
	exists = resp.Success
	return
}
