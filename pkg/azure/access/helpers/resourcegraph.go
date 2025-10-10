// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"fmt"

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
	defer instrument.AZAPIMetricRecorderFn(resourceGraphQueryServiceLabel, &err)()

	query := fmt.Sprintf(queryTemplate, templateArgs...)
	var skipToken *string

	// Continue fetching results while there is a skipToken
	for {
		queryRequest := armresourcegraph.QueryRequest{
			Query:         to.Ptr(query),
			Options:       nil,
			Subscriptions: []*string{to.Ptr(subscriptionID)},
		}

		// Set skipToken in options if present for subsequent pages
		if skipToken != nil {
			queryRequest.Options = &armresourcegraph.QueryRequestOptions{
				SkipToken: skipToken,
			}
		}

		resources, err := client.Resources(ctx, queryRequest, nil)
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

		// Check if there are more pages to fetch and set skipToken for next iteration
		if resources.SkipToken == nil || *resources.SkipToken == "" {
			break
		}
		skipToken = resources.SkipToken
	}

	return results, nil
}
