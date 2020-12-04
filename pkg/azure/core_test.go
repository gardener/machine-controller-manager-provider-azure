/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	mock "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/mock"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("MachineController", func() {

	azureProviderSecret := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientId":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	azureProviderSecretWithoutazureClientID := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientId":       []byte(""),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	azureProviderSecretWithoutazureClientSecret := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientId":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte(""),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	azureProviderSecretWithoutazureTenantID := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientId":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte(""),
	}

	azureProviderSecretWithoutazureSubscriptionID := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientId":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte(""),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	azureProviderSecretWithoutUserData := map[string][]byte{
		"userData":            []byte(""),
		"azureClientId":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	Describe("#Create Machine", func() {

		DescribeTable("##Table",
			func(
				providerSpec *apis.AzureProviderSpec,
				machineRequest *driver.CreateMachineRequest,
				machineResponse *driver.CreateMachineResponse,
				errToHaveOccurred bool,
				errMessage string,
			) {

				// Create the mock controller and the mock clients
				controller := gomock.NewController(GinkgoT())

				mockPluginSPIImpl := mock.NewMockPluginSPIImpl(controller)
				mockDriver := NewAzureDriver(mockPluginSPIImpl, "")
				ctx := context.Background()

				// call setup before the create machine
				mockDriverClients, err := mockPluginSPIImpl.Setup(machineRequest.Secret)

				// Define all the client expectations here and then proceed with the function call
				fakeClients := mockDriverClients.(*mock.AzureDriverClients)
				fakeClients.Subnet.EXPECT().Get(context.Background(),
					providerSpec.ResourceGroup,
					providerSpec.SubnetInfo.VnetName,
					providerSpec.SubnetInfo.SubnetName,
					"").DoAndReturn(network.Subnet{})

				// if there is no variation in the machine class (various scenarios) call the
				// machineRequest.MachineClass = newAzureMachineClass(providerSpec)
				response, err := mockDriver.CreateMachine(ctx, machineRequest)

				if errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(machineResponse.ProviderID).To(Equal(response.ProviderID))
					Expect(machineResponse.NodeName).To(Equal(response.NodeName))
				}
			},

			Entry("#1 Create a simple machine",
				mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.CreateMachineResponse{
					ProviderID: "azure:///westeurope/dummy-machine",
					NodeName:   "dummy-machine",
				},
				false,
				"",
			),
		)
	})
})

func newMachine(name string) *v1alpha1.Machine {
	return &v1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func newAzureMachineClass(azureProviderSpec []byte) *v1alpha1.MachineClass {
	return &v1alpha1.MachineClass{
		ProviderSpec: runtime.RawExtension{
			Raw: azureProviderSpec,
		},
	}
}

func newSecret(azureProviderSecretRaw map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		Data: azureProviderSecretRaw,
	}
}
