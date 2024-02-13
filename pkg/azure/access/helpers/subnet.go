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

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const subnetGetServiceLabel = "subnet_get"

// GetSubnet fetches a Subnet resource given a resourceGroup, virtualNetworkName and subnetName.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func GetSubnet(ctx context.Context, subnetAccess *armnetwork.SubnetsClient, resourceGroup, virtualNetworkName, subnetName string) (subnet *armnetwork.Subnet, err error) {
	var subnetResp armnetwork.SubnetsClientGetResponse
	defer instrument.AZAPIMetricRecorderFn(subnetGetServiceLabel, &err)()

	subnetResp, err = subnetAccess.Get(ctx, resourceGroup, virtualNetworkName, subnetName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to GET Subnet for [resourceGroup: %s, virtualNetworkName: %s, subnetName: %s]", resourceGroup, virtualNetworkName, subnetName)
		return nil, err
	}
	subnet = &subnetResp.Subnet
	return
}
