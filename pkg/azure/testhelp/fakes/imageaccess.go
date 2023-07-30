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

type ImageAccessBuilder struct {
	server          fakecompute.VirtualMachineImagesServer
	clusterState    *ClusterState
	apiBehaviorSpec *APIBehaviorSpec
}

func (b *ImageAccessBuilder) WithClusterState(clusterState *ClusterState) *ImageAccessBuilder {
	b.clusterState = clusterState
	return b
}

func (b *ImageAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *ImageAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

func (b *ImageAccessBuilder) withGet() *ImageAccessBuilder {
	b.server.Get = func(ctx context.Context, location string, publisherName string, offer string, skus string, version string, options *armcompute.VirtualMachineImagesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachineImagesClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.ResourceGroup, to.Ptr(VMImageResourceType), testhelp.AccessMethodGet)
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
		}
		vmImage := b.clusterState.GetVMImage(vmImageSpec)
		if vmImage == nil {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeVMImageNotFound))
			return
		}
		resp.SetResponse(http.StatusOK, armcompute.VirtualMachineImagesClientGetResponse{VirtualMachineImage: *vmImage}, nil)
		return
	}
	return b
}

func (b *ImageAccessBuilder) Build() (*armcompute.VirtualMachineImagesClient, error) {
	b.withGet()
	return armcompute.NewVirtualMachineImagesClient(testhelp.SubscriptionID, azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fakecompute.NewVirtualMachineImagesServerTransport(&b.server),
		},
	})
}
