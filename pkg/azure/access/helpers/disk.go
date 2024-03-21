// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"k8s.io/klog/v2"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	diskDeleteServiceLabel = "disk_delete"
)

// DeleteDisk deletes disk for passed in resourceGroup and diskName.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func DeleteDisk(ctx context.Context, client *armcompute.DisksClient, resourceGroup, diskName string) (err error) {
	defer instrument.AZAPIMetricRecorderFn(diskDeleteServiceLabel, &err)()
	var poller *runtime.Poller[armcompute.DisksClientDeleteResponse]
	poller, err = client.BeginDelete(ctx, resourceGroup, diskName, nil)
	if err != nil {
		// If target Disk is not found then `BeginDelete` will not return any error. This is treated as a NO-OP and a success is returned instead.
		// If this changes incompatibly in the future then we should explicitly handle the NotFound error.
		errors.LogAzAPIError(err, "Failed to trigger Delete of Disk for [resourceGroup: %s, Name: %s]", resourceGroup, diskName)
		return
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Deleting for [resourceGroup: %s, Name: %s]", diskName, resourceGroup)
	}
	klog.Infof("Successfully deleted Disk: %s, for ResourceGroup: %s", diskName, resourceGroup)
	return
}
