// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
)

// FakeResourceGraphClient is a fake implementation of ResourceGraphClient for testing.
type FakeResourceGraphClient struct {
	// Responses is a list of responses to return in sequence for each call to Resources
	Responses []armresourcegraph.ClientResourcesResponse
	// Errors is a list of errors to return in sequence for each call to Resources
	Errors []error
	// CallCount tracks how many times Resources has been called
	CallCount int
	// RecordedRequests stores all query requests made to the client
	RecordedRequests []armresourcegraph.QueryRequest
}

// NewFakeResourceGraphClient creates a new FakeResourceGraphClient for testing.
func NewFakeResourceGraphClient() *FakeResourceGraphClient {
	return &FakeResourceGraphClient{
		Responses:        []armresourcegraph.ClientResourcesResponse{},
		Errors:           []error{},
		RecordedRequests: []armresourcegraph.QueryRequest{},
	}
}

// Resources implements the ResourceGraphClient interface.
// Returns the next response/error in the sequence based on CallCount.
func (f *FakeResourceGraphClient) Resources(_ context.Context, query armresourcegraph.QueryRequest, _ *armresourcegraph.ClientResourcesOptions) (armresourcegraph.ClientResourcesResponse, error) {
	f.RecordedRequests = append(f.RecordedRequests, query)
	index := f.CallCount
	f.CallCount++

	if index < len(f.Errors) && f.Errors[index] != nil {
		return armresourcegraph.ClientResourcesResponse{}, f.Errors[index]
	}

	if index < len(f.Responses) {
		return f.Responses[index], nil
	}

	return armresourcegraph.ClientResourcesResponse{}, nil
}

// AddResponse adds a response to be returned by the fake client.
func (f *FakeResourceGraphClient) AddResponse(response armresourcegraph.ClientResourcesResponse) *FakeResourceGraphClient {
	f.Responses = append(f.Responses, response)
	return f
}

// AddError adds an error to be returned by the fake client.
func (f *FakeResourceGraphClient) AddError(err error) *FakeResourceGraphClient {
	f.Errors = append(f.Errors, err)
	return f
}

// Reset the fake client state.
func (f *FakeResourceGraphClient) Reset() {
	f.Responses = []armresourcegraph.ClientResourcesResponse{}
	f.Errors = []error{}
	f.CallCount = 0
	f.RecordedRequests = []armresourcegraph.QueryRequest{}
}
