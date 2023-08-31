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

package fakes

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	fakeresourcegraph "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"k8s.io/utils/pointer"
)

type ResourceGraphAccessBuilder struct {
	clusterState    *ClusterState
	server          fakeresourcegraph.Server
	apiBehaviorSpec *APIBehaviorSpec
}

func (b *ResourceGraphAccessBuilder) WithClusterState(clusterState *ClusterState) *ResourceGraphAccessBuilder {
	b.clusterState = clusterState
	return b
}

func (b *ResourceGraphAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *ResourceGraphAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

// withResources sets the implementation for `Resources` method. The implementation is not generic and currently only assumes that user will use this fake server method
// to only test queries that are specified in `helpers.resourcegraph` package. If new queries are added then this implementation should be updated.
func (b *ResourceGraphAccessBuilder) withResources() *ResourceGraphAccessBuilder {
	b.server.Resources = func(ctx context.Context, query armresourcegraph.QueryRequest, options *armresourcegraph.ClientResourcesOptions) (resp azfake.Responder[armresourcegraph.ClientResourcesResponse], errResp azfake.ErrorResponder) {
		var resType *ResourceType
		if query.Query != nil {
			resType = getResourceType(*query.Query)
		}
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.ProviderSpec.ResourceGroup, resType, testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		// ------------------------------- NOTE --------------------------------
		// When a non-existent resource group is passed in the resource graph query
		// then it does not error out instead it returns 0 results.
		//-----------------------------------------------------------------------
		var vmNames []string
		if query.Query != nil {
			if resType != nil {
				switch *resType {
				case VirtualMachinesResourceType:
					vmNames = b.clusterState.GetAllExistingVMNames()
				case NetworkInterfacesResourceType:
					vmNames = b.clusterState.ExtractVMNamesFromNICs()
				}
			} else {
				// if there is not resultType this means that the query does not have a filter on resource type
				// in this case we will now search for VM names across all resources that we preserve as part of ClusterState.
				vmNames = b.clusterState.GetAllVMNamesFromMachineResources()
			}
		}
		// create the response
		// currently the fake implementation does not have paging support. This means the Count is also the TotalRecords.
		queryResp := createResourcesResponse(vmNames)
		resp.SetResponse(http.StatusOK, queryResp, nil)
		return
	}
	return b
}

// getResourceType tries to find the table name from the query string. Resource graph defines one table per resource type.
// Unfortunately I could not find a way to parse the KUSTO Query string to extract the table name and therefore string matching is used here. We can change it if we find a better way
func getResourceType(query string) *ResourceType {
	switch {
	case strings.Contains(query, string(VirtualMachinesResourceType)):
		return to.Ptr(VirtualMachinesResourceType)
	case strings.Contains(query, string(NetworkInterfacesResourceType)):
		return to.Ptr(NetworkInterfacesResourceType)
	default:
		return nil
	}
}

func createResourcesResponse(vmNames []string) armresourcegraph.ClientResourcesResponse {
	body := make([]interface{}, 0, len(vmNames))
	if !utils.IsSliceNilOrEmpty(vmNames) {
		for _, vmName := range vmNames {
			entry := make(map[string]interface{})
			entry["name"] = vmName
			body = append(body, entry)
		}
	}
	return armresourcegraph.ClientResourcesResponse{
		QueryResponse: armresourcegraph.QueryResponse{
			Count:           pointer.Int64(int64(len(vmNames))),
			Data:            body,
			ResultTruncated: to.Ptr(armresourcegraph.ResultTruncatedFalse),
			TotalRecords:    pointer.Int64(int64(len(vmNames))),
		},
	}
}

func (b *ResourceGraphAccessBuilder) Build() (*armresourcegraph.Client, error) {
	b.withResources()
	return armresourcegraph.NewClient(azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fakeresourcegraph.NewServerTransport(&b.server),
		},
	})
}
