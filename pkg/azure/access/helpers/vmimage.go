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
