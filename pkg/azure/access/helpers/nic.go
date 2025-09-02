// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

// labels used for recording prometheus metrics
const (
	nicGetServiceLabel    = "nic_get"
	nicDeleteServiceLabel = "nic_delete"
	nicCreateServiceLabel = "nic_create"
)

const (
	defaultDeleteNICTimeout = 10 * time.Minute
	defaultCreateNICTimeout = 15 * time.Minute
)

// DeleteNIC deletes the NIC identified by a resourceGroup and nicName.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func DeleteNIC(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroup, nicName string) (err error) {
	defer instrument.AZAPIMetricRecorderFn(nicDeleteServiceLabel, &err)()

	var poller *runtime.Poller[armnetwork.InterfacesClientDeleteResponse]
	delCtx, cancelFn := context.WithTimeout(ctx, defaultDeleteNICTimeout)
	defer cancelFn()
	poller, err = client.BeginDelete(delCtx, resourceGroup, nicName, nil)
	if err != nil {
		// If target NIC is not found then `BeginDelete` will not return any error. This is treated as a NO-OP and a success is returned instead.
		// If this changes incompatibly in the future then we should explicitly handle the NotFound error.
		errors.LogAzAPIError(err, "Failed to trigger delete of NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
		return
	}
	_, err = poller.PollUntilDone(delCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Deleting of NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
	}
	klog.Infof("Successfully deleted NIC: %s, for ResourceGroup: %s", nicName, resourceGroup)
	return
}

// GetNIC fetches a NIC identified by resourceGroup and nic name.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func GetNIC(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroup, nicName string) (nic *armnetwork.Interface, err error) {
	defer instrument.AZAPIMetricRecorderFn(nicGetServiceLabel, &err)()

	resp, err := client.Get(ctx, resourceGroup, nicName, nil)
	if err != nil {
		if errors.IsNotFoundAzAPIError(err) {
			return nil, nil
		}
		errors.LogAzAPIError(err, "Failed to get NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
		return nil, err
	}
	return &resp.Interface, nil
}

// CreateNIC creates a NIC given the resourceGroup, nic name and NIC creation parameters.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func CreateNIC(ctx context.Context, nicAccess *armnetwork.InterfacesClient, resourceGroup string, nicParams armnetwork.Interface, nicName string) (nic *armnetwork.Interface, err error) {
	defer instrument.AZAPIMetricRecorderFn(nicCreateServiceLabel, &err)()

	var (
		poller       *runtime.Poller[armnetwork.InterfacesClientCreateOrUpdateResponse]
		creationResp armnetwork.InterfacesClientCreateOrUpdateResponse
	)
	createCtx, cancelFn := context.WithTimeout(ctx, defaultCreateNICTimeout)
	defer cancelFn()

	poller, err = nicAccess.BeginCreateOrUpdate(createCtx, resourceGroup, nicName, nicParams, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger create of NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
		return nil, err
	}
	creationResp, err = poller.PollUntilDone(createCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Creation of NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
	}
	nic = &creationResp.Interface
	return
}
