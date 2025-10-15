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

const (
	testSubscriptionID = "test-subscription-id"
	testQuery          = "Resources | where type =~ 'microsoft.compute/virtualmachines'"
)

// A test struct to use with QueryAndMap
type TestVM struct {
	Name       string
	ResourceID string
	Location   string
}

// Create a test mapper that maps to TestVM
func testVMMapper(row map[string]any) *TestVM {
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

// Create test data
func createTestData(count int, prefix string) []any {
	data := make([]any, count)
	for i := 0; i < count; i++ {
		data[i] = map[string]any{
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

	results, err := QueryAndMap(ctx, fakeClient, testSubscriptionID, testVMMapper, testQuery)

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

	results, err := QueryAndMap(ctx, fakeClient, testSubscriptionID, testVMMapper, testQuery)

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
				Data:         nil,
				SkipToken:    nil,
			},
		})

	results, err := QueryAndMap(ctx, fakeClient, testSubscriptionID, testVMMapper, testQuery)

	g.Expect(err).To(BeNil())
	g.Expect(len(results)).To(Equal(0))
	g.Expect(fakeClient.CallCount).To(Equal(1))
}

func TestQueryAndMap_APIError(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	fakeClient := NewFakeResourceGraphClient().AddError(errors.New("API call failed: AuthorizationFailed"))

	results, err := QueryAndMap(ctx, fakeClient, testSubscriptionID, testVMMapper, testQuery)

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

	results, err := QueryAndMap(ctx, fakeClient, testSubscriptionID, testVMMapper, testQuery)

	g.Expect(err).NotTo(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("network error during pagination"))
	g.Expect(results).To(BeNil())
	g.Expect(fakeClient.CallCount).To(Equal(2))
}

func TestMapper_IncompleteData(t *testing.T) {
	g := NewWithT(t)

	testCases := []struct {
		name          string
		row           map[string]any
		isMissingData bool
	}{
		{
			name: "missing name field",
			row: map[string]any{
				"id":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
				"location": "westeurope",
			},
			isMissingData: true,
		},
		{
			name: "missing id field",
			row: map[string]any{
				"name":     "test-vm-1",
				"location": "westeurope",
			},
			isMissingData: true,
		},
		{
			name: "missing location field",
			row: map[string]any{
				"name": "test-vm-1",
				"id":   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
			},
			isMissingData: true,
		},
		{
			name: "wrong type for name",
			row: map[string]any{
				"name":     123,
				"id":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
				"location": "westeurope",
			},
			isMissingData: true,
		},
		{
			name: "all fields present and correct",
			row: map[string]any{
				"name":     "test-vm-1",
				"id":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
				"location": "westeurope",
			},
			isMissingData: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			result := testVMMapper(tc.row)
			if tc.isMissingData {
				g.Expect(result).To(BeNil())
			} else {
				g.Expect(result).NotTo(BeNil())
				g.Expect(result.Name).To(Equal("test-vm-1"))
			}
		})
	}
}

func TestEmptySkipToken(t *testing.T) {
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

	results, err := QueryAndMap(ctx, fakeClient, testSubscriptionID, testVMMapper, testQuery)

	g.Expect(err).To(BeNil())
	g.Expect(len(results)).To(Equal(2))
	g.Expect(fakeClient.CallCount).To(Equal(1)) // Should not try to fetch more pages
}
