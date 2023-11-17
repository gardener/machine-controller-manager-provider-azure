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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
	"k8s.io/utils/pointer"
)

const (
	resourceGraphQueryServiceLabel = "resource_graph_query"
)

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
