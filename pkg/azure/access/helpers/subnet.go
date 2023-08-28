package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

// GetSubnet fetches a Subnet resource given a resourceGroup, virtualNetworkName and subnetName.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func GetSubnet(ctx context.Context, subnetAccess *armnetwork.SubnetsClient, resourceGroup, virtualNetworkName, subnetName string) (subnet *armnetwork.Subnet, err error) {
	var subnetResp armnetwork.SubnetsClientGetResponse
	defer instrument.RecordAzAPIMetric(err, subnetGetServiceLabel, time.Now())
	subnetResp, err = subnetAccess.Get(ctx, resourceGroup, virtualNetworkName, subnetName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to GET Subnet for [resourceGroup: %s, virtualNetworkName: %s, subnetName: %s]", resourceGroup, virtualNetworkName, subnetName)
		return nil, err
	}
	subnet = &subnetResp.Subnet
	return
}
