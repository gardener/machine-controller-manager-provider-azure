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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
)

// ImageAccessBuilder is a builder for VM images access.
type ImageAccessBuilder struct {
	server          fakecompute.VirtualMachineImagesServer
	clusterState    *ClusterState
	apiBehaviorSpec *APIBehaviorSpec
}

// WithClusterState initializes builder with a ClusterState.
func (b *ImageAccessBuilder) WithClusterState(clusterState *ClusterState) *ImageAccessBuilder {
	b.clusterState = clusterState
	return b
}

// WithAPIBehaviorSpec initializes the builder with a APIBehaviorSpec.
func (b *ImageAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *ImageAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

// withGet implements the Get method of armcompute.VirtualMachineImagesClient and initializes the backing fake server's Get method with the anonymous function implementation.
func (b *ImageAccessBuilder) withGet() *ImageAccessBuilder {
	b.server.Get = func(ctx context.Context, location string, publisherName string, offer string, skus string, version string, options *armcompute.VirtualMachineImagesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachineImagesClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.ProviderSpec.ResourceGroup, to.Ptr(VMImageResourceType), testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		vmImageSpec := VMImageSpec{
			Publisher: publisherName,
			Offer:     offer,
			SKU:       skus,
			Version:   version,
			OfferType: defaultOfferType,
		}
		vmImage := b.clusterState.GetVirtualMachineImage(vmImageSpec)
		if vmImage == nil {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeVMImageNotFound))
			return
		}
		resp.SetResponse(http.StatusOK, armcompute.VirtualMachineImagesClientGetResponse{VirtualMachineImage: *vmImage}, nil)
		return
	}
	return b
}

// Build builds the armcompute.VirtualMachineImagesClient.
func (b *ImageAccessBuilder) Build() (*armcompute.VirtualMachineImagesClient, error) {
	b.withGet()
	return armcompute.NewVirtualMachineImagesClient(testhelp.SubscriptionID, &azfake.TokenCredential{}, &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fakecompute.NewVirtualMachineImagesServerTransport(&b.server),
		},
	})
}
