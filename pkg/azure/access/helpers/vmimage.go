// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const vmImageGetServiceLabel = "virtual_machine_image_get"

// GetVMImage fetches the VM Image given a location and image reference.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func GetVMImage(ctx context.Context, vmImagesAccess *armcompute.VirtualMachineImagesClient, location string, imageRef armcompute.ImageReference) (vmImage *armcompute.VirtualMachineImage, err error) {
	defer instrument.AZAPIMetricRecorderFn(vmImageGetServiceLabel, &err)()

	resp, err := vmImagesAccess.Get(ctx, location, *imageRef.Publisher, *imageRef.Offer, *imageRef.SKU, *imageRef.Version, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to get VM Image [Location: %s, ImageRef: %+v]", location, imageRef)
		return nil, err
	}
	return &resp.VirtualMachineImage, nil
}
