package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	diskDeleteServiceLabel = "disk_delete"
)

func DeleteDiskIfExists(ctx context.Context, client *armcompute.DisksClient, resourceGroup, diskName string) (err error) {
	return
}

func deleteDisk(ctx context.Context, client *armcompute.DisksClient, resourceGroup, diskName string, pollInterval time.Duration) (err error) {
	defer instrument.RecordAzAPIMetric(err, diskDeleteServiceLabel, time.Now())
	var poller *runtime.Poller[armcompute.DisksClientDeleteResponse]
	poller, err = client.BeginDelete(ctx, resourceGroup, diskName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger Delete of Disk for [resourceGroup: %s, diskName: %s]", resourceGroup, diskName)
		return
	}
	_, err = poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: pollInterval})
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Deleting of Disk: %s for ResourceGroup: %s", diskName, resourceGroup)
	}
	return
}
