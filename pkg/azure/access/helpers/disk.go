package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	diskDeleteServiceLabel = "disk_delete"
)

func DeleteDisk(ctx context.Context, client *armcompute.DisksClient, resourceGroup, diskName string) (err error) {
	defer instrument.RecordAzAPIMetric(err, diskDeleteServiceLabel, time.Now())
	var poller *runtime.Poller[armcompute.DisksClientDeleteResponse]
	poller, err = client.BeginDelete(ctx, resourceGroup, diskName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger Delete of Disk for [resourceGroup: %s, Name: %s]", resourceGroup, diskName)
		return
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Deleting for [resourceGroup: %s, Name: %s]", diskName, resourceGroup)
	}
	return
}
