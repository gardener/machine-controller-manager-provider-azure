// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	fakenetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
)

// NICAccessBuilder is a builder for NIC access.
type NICAccessBuilder struct {
	clusterState    *ClusterState
	server          fakenetwork.InterfacesServer
	apiBehaviorSpec *APIBehaviorSpec
}

// WithClusterState initializes builder with a ClusterState.
func (b *NICAccessBuilder) WithClusterState(clusterState *ClusterState) *NICAccessBuilder {
	b.clusterState = clusterState
	return b
}

// WithAPIBehaviorSpec initializes the builder with a APIBehaviorSpec.
func (b *NICAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *NICAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

// withGet implements the Get method of armnetwork.InterfacesClient and initializes the backing fake server's Get method with the anonymous function implementation.
func (b *NICAccessBuilder) withGet() *NICAccessBuilder {
	b.server.Get = func(ctx context.Context, resourceGroupName string, nicName string, _ *armnetwork.InterfacesClientGetOptions) (resp azfake.Responder[armnetwork.InterfacesClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, nicName, testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}
		nic := b.clusterState.GetNIC(nicName)
		if nic == nil {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceNotFound))
			return
		}
		nicResponse := armnetwork.InterfacesClientGetResponse{Interface: *nic}
		resp.SetResponse(http.StatusOK, nicResponse, nil)
		return
	}
	return b
}

// withBeginDelete implements the BeingDelete method of armnetwork.InterfacesClient and initializes the backing fake server's BeginDelete method with the anonymous function implementation.
func (b *NICAccessBuilder) withBeginDelete() *NICAccessBuilder {
	b.server.BeginDelete = func(ctx context.Context, resourceGroupName string, nicName string, _ *armnetwork.InterfacesClientBeginDeleteOptions) (resp azfake.PollerResponder[armnetwork.InterfacesClientDeleteResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, nicName, testhelp.AccessMethodBeginDelete)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}
		// Azure API NIC deletion does not fail if the NIC does not exist. It still returns 200 Ok.
		nic := b.clusterState.GetNIC(nicName)
		if nic != nil && nic.Properties.VirtualMachine != nil && !utils.IsNilOrEmptyStringPtr(nic.Properties.VirtualMachine.ID) {
			errResp.SetError(testhelp.ConflictErr(testhelp.ErrorCodeOperationNotAllowed))
			return
		}
		b.clusterState.DeleteNIC(nicName)
		resp.SetTerminalResponse(200, armnetwork.InterfacesClientDeleteResponse{}, nil)
		return
	}
	return b
}

// withBeginCreateOrUpdate implements the BeginCreateOrUpdate method of armnetwork.InterfacesClient and initializes the backing fake server's BeginCreateOrUpdate method with the anonymous function implementation.
func (b *NICAccessBuilder) withBeginCreateOrUpdate() *NICAccessBuilder {
	b.server.BeginCreateOrUpdate = func(ctx context.Context, resourceGroupName string, nicName string, parameters armnetwork.Interface, _ *armnetwork.InterfacesClientBeginCreateOrUpdateOptions) (resp azfake.PollerResponder[armnetwork.InterfacesClientCreateOrUpdateResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResource(ctx, resourceGroupName, nicName, testhelp.AccessMethodBeginCreateOrUpdate)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		if b.clusterState.ProviderSpec.ResourceGroup != resourceGroupName {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceGroupNotFound))
			return
		}
		nic := b.clusterState.CreateNIC(nicName, &parameters)
		resp.SetTerminalResponse(http.StatusOK, armnetwork.InterfacesClientCreateOrUpdateResponse{Interface: *nic}, nil)
		return
	}
	return b
}

// Build builds armnetwork.InterfacesClient.
func (b *NICAccessBuilder) Build() (*armnetwork.InterfacesClient, error) {
	b.withGet().withBeginDelete().withBeginCreateOrUpdate()
	return armnetwork.NewInterfacesClient(testhelp.SubscriptionID, &azfake.TokenCredential{}, &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakenetwork.NewInterfacesServerTransport(&b.server),
		},
	})
}
