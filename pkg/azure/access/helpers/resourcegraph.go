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
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

const (
	listVMsQueryTemplate = `
	Resources
	| where type =~ 'Microsoft.Compute/virtualMachines'
	| where resourceGroup =~ '%s'
	| extend tagKeys = bag_keys(tags)
	| where tagKeys hasprefix "kubernetes.io-cluster-" and tagKeys hasprefix "kubernetes.io-role-"
	| project name
	`
	listNICsQueryTemplate = `
	Resources
	| where type =~ 'microsoft.network/networkinterfaces'
	| where resourceGroup =~ '%s'
	| extend tagKeys = bag_keys(tags)
	| where tagKeys hasprefix "kubernetes.io-cluster-" and tagKeys hasprefix "kubernetes.io-role-"
	| project name
	`
	nicSuffix                      = "-nic"
	resourceGraphQueryServiceLabel = "resource_graph_query"
)

// ExtractVMNamesFromVirtualMachinesAndNICs extracts VM names from virtual machines and NIC names and returns a slice of unique vm names.
func ExtractVMNamesFromVirtualMachinesAndNICs(ctx context.Context, client *armresourcegraph.Client, subscriptionID, resourceGroup string) ([]string, error) {
	vmNames := sets.New[string]()
	vmNamesFromVirtualMachines, err := QueryAndMap[string](ctx, client, subscriptionID, createVMNameMapperFn(nil), listVMsQueryTemplate, resourceGroup)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get VM names from VirtualMachines for resourceGroup :%s: error: %v", resourceGroup, err))
	}
	vmNames.Insert(vmNamesFromVirtualMachines...)

	// extract VM Names from existing NICs. Why is this required?
	// A Machine in MCM terminology is a collective entity consisting of but not limited to VM, NIC(s), Disk(s).
	// MCM orphan collection needs to track resources which have a separate lifecycle (currently in case of azure it is VM's and NICs.
	// Disks (OS and Data) are created and deleted along with then VM.) and which are now orphaned. Unfortunately, MCM only orphan collects
	// machines (a collective resource) and a machine is uniquely identified by a VM name (again not so ideal).
	// In order to get any orphaned VM or NIC, its currently essential that a VM name which serves as a unique machine name should be collected
	// by introspecting VMs and NICs. Ideally you would change the response struct to separately capture VM name(s) and NIC name(s) under MachineInfo
	// and have a slice of such MachineInfo returned as part of this provider method.
	vmNamesFromNICs, err := QueryAndMap[string](ctx, client, subscriptionID, createVMNameMapperFn(to.Ptr(nicSuffix)), listNICsQueryTemplate, resourceGroup)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get VM names from NICs for resourceGroup :%s: error: %v", resourceGroup, err))
	}
	vmNames.Insert(vmNamesFromNICs...)
	return vmNames.UnsortedList(), nil
}

// MapperFn maps a row of result (represented as map[string]interface{}) to any type T.
type MapperFn[T any] func(map[string]interface{}) *T

// QueryAndMap fires a resource graph KUSTO query constructing it from queryTemplate and templateArgs.
// The result of the query are then mapped using a mapperFn and the result or an error is returned.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func QueryAndMap[T any](ctx context.Context, client *armresourcegraph.Client, subscriptionID string, mapperFn MapperFn[T], queryTemplate string, templateArgs ...any) (results []T, err error) {
	defer instrument.RecordAzAPIMetric(err, resourceGraphQueryServiceLabel, time.Now())
	query := fmt.Sprintf(queryTemplate, templateArgs...)
	resources, err := client.Resources(ctx,
		armresourcegraph.QueryRequest{
			Query:         to.Ptr(query),
			Options:       nil,
			Subscriptions: []*string{to.Ptr(subscriptionID)},
		}, nil)

	if err != nil {
		errors.LogAzAPIError(err, "ResourceGraphQuery failure to execute Query: %s", query)
		return nil, err
	}

	if resources.TotalRecords == pointer.Int64(0) {
		return results, nil
	}

	// resourceResponse.Data is a []interface{}
	if objSlice, ok := resources.Data.([]interface{}); ok {
		for _, obj := range objSlice {
			// Each obj in resourceResponse.Data is a map[string]Interface{}
			rowElements := obj.(map[string]interface{})
			result := mapperFn(rowElements)
			if result != nil {
				results = append(results, *result)
			}
		}
	}

	return
}

func createVMNameMapperFn(suffix *string) MapperFn[string] {
	return func(r map[string]interface{}) *string {
		if resourceNameVal, keyFound := r["name"]; keyFound {
			resourceName := resourceNameVal.(string)
			if suffix != nil && strings.HasSuffix(resourceName, *suffix) {
				return to.Ptr(resourceName[:len(resourceName)-len(*suffix)])
			}
			return to.Ptr(resourceName)
		}
		return nil
	}
}
