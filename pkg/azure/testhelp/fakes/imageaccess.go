// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
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
	b.server.Get = func(ctx context.Context, _ string, publisherName string, offer string, skus string, version string, _ *armcompute.VirtualMachineImagesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachineImagesClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.ProviderSpec.ResourceGroup, to.Ptr(utils.VMImageResourceType), testhelp.AccessMethodGet)
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
