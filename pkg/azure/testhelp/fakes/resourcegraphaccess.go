// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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

// ResourceGraphAccessBuilder is a builder for Resource Graph access.
type ResourceGraphAccessBuilder struct {
	clusterState    *ClusterState
	server          fakeresourcegraph.Server
	apiBehaviorSpec *APIBehaviorSpec
}

// WithClusterState initializes builder with a ClusterState.
func (b *ResourceGraphAccessBuilder) WithClusterState(clusterState *ClusterState) *ResourceGraphAccessBuilder {
	b.clusterState = clusterState
	return b
}

// WithAPIBehaviorSpec initializes the builder with a APIBehaviorSpec.
func (b *ResourceGraphAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *ResourceGraphAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

// withResources sets the implementation for `Resources` method. The implementation is not generic and currently only assumes that user will use this fake server method
// to only test queries that are specified in `helpers.ExtractVMNamesFromVMsNICsDisks` function. If new queries are added then this implementation should be updated.
func (b *ResourceGraphAccessBuilder) withResources() *ResourceGraphAccessBuilder {
	b.server.Resources = func(ctx context.Context, query armresourcegraph.QueryRequest, _ *armresourcegraph.ClientResourcesOptions) (resp azfake.Responder[armresourcegraph.ClientResourcesResponse], errResp azfake.ErrorResponder) {

		foundResourceTypes := getResourceTypes(query)

		if b.apiBehaviorSpec != nil && !utils.IsSliceNilOrEmpty(foundResourceTypes) {
			for _, resType := range foundResourceTypes {
				err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.ProviderSpec.ResourceGroup, &resType, testhelp.AccessMethodResources)
				if err != nil {
					errResp.SetError(err)
					return
				}
			}
		}
		// ------------------------------- NOTE --------------------------------
		// When a non-existent resource group is passed in the resource graph query
		// then it does not error out instead it returns 0 results.
		//-----------------------------------------------------------------------
		resTypeToVMNames := make(map[string][]string)
		if query.Query != nil {
			tagsToMatch := b.getProviderSpecTagKeysToMatch()
			for _, resType := range foundResourceTypes {
				switch resType {
				case utils.VirtualMachinesResourceType:
					vmNames := b.clusterState.GetVMsMatchingTagKeys(tagsToMatch)
					if !utils.IsSliceNilOrEmpty(vmNames) {
						resTypeToVMNames[string(resType)] = vmNames
					}
				case utils.NetworkInterfacesResourceType:
					vmNames := b.clusterState.GetNICNamesMatchingTagKeys(tagsToMatch)
					if !utils.IsSliceNilOrEmpty(vmNames) {
						resTypeToVMNames[string(resType)] = vmNames
					}
				case utils.DiskResourceType:
					vmNames := b.clusterState.GetDiskNamesMatchingTagKeys(tagsToMatch)
					if !utils.IsSliceNilOrEmpty(vmNames) {
						resTypeToVMNames[string(resType)] = vmNames
					}
				}
			}
		}
		// create the response
		// currently the fake implementation does not have paging support. This means the Count is also the TotalRecords.
		queryResp := createResourcesResponse(resTypeToVMNames)
		resp.SetResponse(http.StatusOK, queryResp, nil)
		return
	}
	return b
}

// getResourceTypes tries to find the table names from the query string. Resource graph defines one table per resource type.
// Unfortunately there seems to be no way to parse the KUSTO Query string to extract the table name and therefore string matching is used here.
// We can change it if we find a better way
func getResourceTypes(query armresourcegraph.QueryRequest) []utils.ResourceType {
	var foundResourceTypes []utils.ResourceType
	if query.Query == nil {
		return foundResourceTypes
	}
	resourceTypesToMatch := []utils.ResourceType{utils.VirtualMachinesResourceType, utils.NetworkInterfacesResourceType, utils.DiskResourceType}
	for _, resType := range resourceTypesToMatch {
		if strings.Contains(*query.Query, string(resType)) {
			foundResourceTypes = append(foundResourceTypes, resType)
		}
	}
	return foundResourceTypes
}

func (b *ResourceGraphAccessBuilder) getProviderSpecTagKeysToMatch() []string {
	tagKeys := make([]string, 0, 2)
	for k := range b.clusterState.ProviderSpec.Tags {
		if strings.HasPrefix(k, utils.ClusterTagPrefix) || strings.HasPrefix(k, utils.RoleTagPrefix) {
			tagKeys = append(tagKeys, k)
		}
	}
	return tagKeys
}

func createResourcesResponse(resTypeToVMNames map[string][]string) armresourcegraph.ClientResourcesResponse {
	body := make([]interface{}, 0, len(resTypeToVMNames))
	for resType, vmNames := range resTypeToVMNames {
		for _, vmName := range vmNames {
			entry := make(map[string]interface{})
			entry["type"] = resType
			entry["name"] = vmName
			body = append(body, entry)
		}
	}
	return armresourcegraph.ClientResourcesResponse{
		QueryResponse: armresourcegraph.QueryResponse{
			Count:           pointer.Int64(int64(len(resTypeToVMNames))),
			Data:            body,
			ResultTruncated: to.Ptr(armresourcegraph.ResultTruncatedFalse),
			TotalRecords:    pointer.Int64(int64(len(resTypeToVMNames))),
		},
	}
}

// Build builds armresourcegraph.Client.
func (b *ResourceGraphAccessBuilder) Build() (*armresourcegraph.Client, error) {
	b.withResources()
	return armresourcegraph.NewClient(&azfake.TokenCredential{}, &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fakeresourcegraph.NewServerTransport(&b.server),
		},
	})
}
