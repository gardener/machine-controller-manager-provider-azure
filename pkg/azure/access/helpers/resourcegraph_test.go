// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	. "github.com/onsi/gomega"
)

// A test struct to use with QueryAndMap
type TestVM struct {
	Name       string
	ResourceID string
	Location   string
}

// Create a test mapper that maps to TestVM
func testVMMapper(row map[string]interface{}) *TestVM {
	name, nameOk := row["name"].(string)
	resourceID, idOk := row["id"].(string)
	location, locOk := row["location"].(string)

	if !nameOk || !idOk || !locOk {
		return nil
	}

	return &TestVM{
		Name:       name,
		ResourceID: resourceID,
		Location:   location,
	}
}

// Create a test mapper that returns nil (helper function to test filtering)
func nilReturningMapper(_ map[string]interface{}) *TestVM {
	return nil
}

// Create test data
func createTestData(count int, prefix string) []interface{} {
	data := make([]interface{}, count)
	for i := 0; i < count; i++ {
		data[i] = map[string]interface{}{
			"name":     prefix + "-vm-" + string(rune('0'+i)),
			"id":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/" + prefix + "-vm-" + string(rune('0'+i)),
			"location": "westeurope",
		}
	}
	return data
}

func TestQueryAndMap_Success_SinglePage(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	testData := createTestData(3, "test")
	fakeClient := NewFakeResourceGraphClient().
		AddResponse(armresourcegraph.ClientResourcesResponse{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](3),
				Data:         testData,
				SkipToken:    nil,
			},
		})

	results, err := QueryAndMap(ctx, fakeClient, "test-subscription-id", testVMMapper, "Resources | where type =~ 'microsoft.compute/virtualmachines'")

	g.Expect(err).To(BeNil())
	g.Expect(len(results)).To(Equal(3))
	g.Expect(fakeClient.CallCount).To(Equal(1))

	g.Expect(results[0].Name).To(ContainSubstring("test-vm"))
	g.Expect(results[0].Location).To(Equal("westeurope"))
}

func TestQueryAndMap_Success_MultiplePages(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	page1Data := createTestData(2, "page1")
	page2Data := createTestData(2, "page2")
	page3Data := createTestData(1, "page3")

	fakeClient := NewFakeResourceGraphClient().
		AddResponse(armresourcegraph.ClientResourcesResponse{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](2),
				Data:         page1Data,
				SkipToken:    to.Ptr("token-page2"),
			},
		}).
		AddResponse(armresourcegraph.ClientResourcesResponse{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](2),
				Data:         page2Data,
				SkipToken:    to.Ptr("token-page3"),
			},
		}).
		AddResponse(armresourcegraph.ClientResourcesResponse{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](1),
				Data:         page3Data,
				SkipToken:    nil,
			},
		})

	results, err := QueryAndMap(ctx, fakeClient, "test-subscription-id", testVMMapper, "Resources | where type =~ 'microsoft.compute/virtualmachines'")

	g.Expect(err).To(BeNil())
	g.Expect(len(results)).To(Equal(5))
	g.Expect(fakeClient.CallCount).To(Equal(3))

	g.Expect(len(fakeClient.RecordedRequests)).To(Equal(3))
	g.Expect(fakeClient.RecordedRequests[0].Options).To(BeNil())
	g.Expect(fakeClient.RecordedRequests[1].Options).NotTo(BeNil())
	g.Expect(*fakeClient.RecordedRequests[1].Options.SkipToken).To(Equal("token-page2"))
	g.Expect(fakeClient.RecordedRequests[2].Options).NotTo(BeNil())
	g.Expect(*fakeClient.RecordedRequests[2].Options.SkipToken).To(Equal("token-page3"))
}

func TestQueryAndMap_NoResults(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	fakeClient := NewFakeResourceGraphClient().
		AddResponse(armresourcegraph.ClientResourcesResponse{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](0),
				Data:         []interface{}{},
				SkipToken:    nil,
			},
		})

	results, err := QueryAndMap(ctx, fakeClient, "test-subscription-id", testVMMapper, "Resources | where type =~ 'microsoft.compute/virtualmachines'")

	g.Expect(err).To(BeNil())
	g.Expect(len(results)).To(Equal(0))
	g.Expect(fakeClient.CallCount).To(Equal(1))
}

func TestQueryAndMap_APIError(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	fakeClient := NewFakeResourceGraphClient().AddError(errors.New("API call failed: AuthorizationFailed"))

	results, err := QueryAndMap(ctx, fakeClient, "test-subscription-id", testVMMapper, "Resources | where type =~ 'microsoft.compute/virtualmachines'")

	g.Expect(err).NotTo(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("API call failed"))
	g.Expect(results).To(BeNil())
	g.Expect(fakeClient.CallCount).To(Equal(1))
}

func TestQueryAndMap_ErrorInMiddleOfPagination(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	testData := createTestData(2, "page1")
	testError := errors.New("network error during pagination")

	fakeClient := NewFakeResourceGraphClient()
	fakeClient.Responses = []armresourcegraph.ClientResourcesResponse{
		{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](2),
				Data:         testData,
				SkipToken:    to.Ptr("token-page2"),
			},
		},
		{}, // Empty response for error
	}
	fakeClient.Errors = []error{
		nil,
		testError,
	}

	results, err := QueryAndMap(ctx, fakeClient, "test-subscription-id", testVMMapper, "Resources | where type =~ 'microsoft.compute/virtualmachines'")

	g.Expect(err).NotTo(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("network error during pagination"))
	g.Expect(results).To(BeNil())
	g.Expect(fakeClient.CallCount).To(Equal(2))
}

func TestMapper_IncompleteData(t *testing.T) {
	g := NewWithT(t)

	testCases := []struct {
		name        string
		row         map[string]interface{}
		shouldBeNil bool
	}{
		{
			name: "missing name field",
			row: map[string]interface{}{
				"id":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
				"location": "westeurope",
			},
			shouldBeNil: true,
		},
		{
			name: "missing id field",
			row: map[string]interface{}{
				"name":     "test-vm-1",
				"location": "westeurope",
			},
			shouldBeNil: true,
		},
		{
			name: "missing location field",
			row: map[string]interface{}{
				"name": "test-vm-1",
				"id":   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
			},
			shouldBeNil: true,
		},
		{
			name: "wrong type for name",
			row: map[string]interface{}{
				"name":     123,
				"id":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
				"location": "westeurope",
			},
			shouldBeNil: true,
		},
		{
			name: "all fields present and correct",
			row: map[string]interface{}{
				"name":     "test-vm-1",
				"id":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
				"location": "westeurope",
			},
			shouldBeNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			result := testVMMapper(tc.row)
			if tc.shouldBeNil {
				g.Expect(result).To(BeNil())
			} else {
				g.Expect(result).NotTo(BeNil())
				g.Expect(result.Name).To(Equal("test-vm-1"))
			}
		})
	}
}

func TestEmptySkipToken(t *testing.T) { //CHECK
	g := NewWithT(t)
	ctx := context.Background()

	testData := createTestData(2, "test")

	fakeClient := NewFakeResourceGraphClient().
		AddResponse(armresourcegraph.ClientResourcesResponse{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](2),
				Data:         testData,
				SkipToken:    to.Ptr(""),
			},
		})

	results, err := QueryAndMap(ctx, fakeClient, "test-subscription-id", testVMMapper, "Resources | where type =~ 'microsoft.compute/virtualmachines'")

	g.Expect(err).To(BeNil())
	g.Expect(len(results)).To(Equal(2))
	g.Expect(fakeClient.CallCount).To(Equal(1)) // Should not try to fetch more pages
}

func TestQueryAndMap_WithMapperFiltering(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	testData := createTestData(5, "test")

	fakeClient := NewFakeResourceGraphClient().
		AddResponse(armresourcegraph.ClientResourcesResponse{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](5),
				Data:         testData,
				SkipToken:    nil,
			},
		})

	results, err := QueryAndMap(ctx, fakeClient, "test-subscription-id", nilReturningMapper, "Resources | where type =~ 'microsoft.compute/virtualmachines'")

	g.Expect(err).To(BeNil())
	g.Expect(len(results)).To(Equal(0)) // All results filtered out by mapper
	g.Expect(fakeClient.CallCount).To(Equal(1))
}

func TestQueryAndMap_WithQueryTemplateAndArgs(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	testData := createTestData(2, "filtered")

	fakeClient := NewFakeResourceGraphClient().
		AddResponse(armresourcegraph.ClientResourcesResponse{
			QueryResponse: armresourcegraph.QueryResponse{
				TotalRecords: to.Ptr[int64](2),
				Data:         testData,
				SkipToken:    nil,
			},
		})

	// Test with query template and arguments
	queryTemplate := "Resources | where type =~ 'microsoft.compute/virtualmachines' and resourceGroup =~ '%s'"
	results, err := QueryAndMap(ctx, fakeClient, "test-subscription-id", testVMMapper, queryTemplate, "test-resource-group")

	g.Expect(err).To(BeNil())
	g.Expect(len(results)).To(Equal(2))

	g.Expect(len(fakeClient.RecordedRequests)).To(Equal(1))
	g.Expect(*fakeClient.RecordedRequests[0].Query).To(ContainSubstring("test-resource-group"))
}

func TestFakeResourceGraphClient_Reset(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	fakeClient := NewFakeResourceGraphClient()

	testData := createTestData(1, "test")
	fakeClient.AddResponse(armresourcegraph.ClientResourcesResponse{
		QueryResponse: armresourcegraph.QueryResponse{
			TotalRecords: to.Ptr[int64](1),
			Data:         testData,
			SkipToken:    nil,
		},
	})

	_, _ = QueryAndMap(ctx, fakeClient, "test-sub", testVMMapper, "test query")

	g.Expect(fakeClient.CallCount).To(Equal(1))
	g.Expect(len(fakeClient.RecordedRequests)).To(Equal(1))

	fakeClient.Reset()

	g.Expect(fakeClient.CallCount).To(Equal(0))
	g.Expect(len(fakeClient.RecordedRequests)).To(Equal(0))
	g.Expect(len(fakeClient.Responses)).To(Equal(0))
	g.Expect(len(fakeClient.Errors)).To(Equal(0))
}
