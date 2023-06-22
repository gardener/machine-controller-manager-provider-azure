package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	diskDeleteServiceLabel = "disk_delete"
	diskGetServiceLabel    = "disk_get"
)

func DeleteDiskIfExists(ctx context.Context, client *armcompute.DisksClient, resourceGroup, diskName string) (err error) {
	disk, err := GetDisk(ctx, client, resourceGroup, diskName)
	if err != nil {
		return err
	}
	if disk.ManagedBy != nil {
		return fmt.Errorf("cannot delete Disk [ResourceGroup: %s, Name: %s] as it is still associated with the VM: %s", resourceGroup, diskName, *disk.ManagedBy)
	}
	return deleteDisk(ctx, client, resourceGroup, diskName)
}

func GetDisk(ctx context.Context, client *armcompute.DisksClient, resourceGroup, diskName string) (disk *armcompute.Disk, err error) {
	defer instrument.RecordAzAPIMetric(err, diskGetServiceLabel, time.Now())
	resp, err := client.Get(ctx, resourceGroup, diskName, nil)
	if err != nil {
		if errors.IsNotFoundAzAPIError(err) {
			return nil, nil
		}
		errors.LogAzAPIError(err, "Failed to get Disk [ResourceGroup: %s, Name: %s]", resourceGroup, diskName)
		return nil, err
	}
	return &resp.Disk, nil
}

func deleteDisk(ctx context.Context, client *armcompute.DisksClient, resourceGroup, diskName string) (err error) {
	defer instrument.RecordAzAPIMetric(err, diskDeleteServiceLabel, time.Now())
	var poller *runtime.Poller[armcompute.DisksClientDeleteResponse]
	poller, err = client.BeginDelete(ctx, resourceGroup, diskName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger Delete of Disk for [resourceGroup: %s, diskName: %s]", resourceGroup, diskName)
		return
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Deleting of Disk: %s for ResourceGroup: %s", diskName, resourceGroup)
	}
	return
}
