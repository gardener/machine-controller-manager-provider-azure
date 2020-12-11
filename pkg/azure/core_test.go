/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package azure

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"

	apis "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
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

	// azureProviderSecretWithoutazureClientID := map[string][]byte{
	// 	"userData":            []byte("dummy-data"),
	// 	"azureClientId":       []byte(""),
	// 	"azureClientSecret":   []byte("dummy-client-secret"),
	// 	"azureSubscriptionId": []byte("dummy-subcription-id"),
	// 	"azureTenantId":       []byte("dummy-tenant-id"),
	// }

	// azureProviderSecretWithoutazureClientSecret := map[string][]byte{
	// 	"userData":            []byte("dummy-data"),
	// 	"azureClientId":       []byte("dummy-client-id"),
	// 	"azureClientSecret":   []byte(""),
	// 	"azureSubscriptionId": []byte("dummy-subcription-id"),
	// 	"azureTenantId":       []byte("dummy-tenant-id"),
	// }

	// azureProviderSecretWithoutazureTenantID := map[string][]byte{
	// 	"userData":            []byte("dummy-data"),
	// 	"azureClientId":       []byte("dummy-client-id"),
	// 	"azureClientSecret":   []byte("dummy-client-secret"),
	// 	"azureSubscriptionId": []byte("dummy-subcription-id"),
	// 	"azureTenantId":       []byte(""),
	// }

	// azureProviderSecretWithoutazureSubscriptionID := map[string][]byte{
	// 	"userData":            []byte("dummy-data"),
	// 	"azureClientId":       []byte("dummy-client-id"),
	// 	"azureClientSecret":   []byte("dummy-client-secret"),
	// 	"azureSubscriptionId": []byte(""),
	// 	"azureTenantId":       []byte("dummy-tenant-id"),
	// }

	// azureProviderSecretWithoutUserData := map[string][]byte{
	// 	"userData":            []byte(""),
	// 	"azureClientId":       []byte("dummy-client-id"),
	// 	"azureClientSecret":   []byte("dummy-client-secret"),
	// 	"azureSubscriptionId": []byte("dummy-subcription-id"),
	// 	"azureTenantId":       []byte("dummy-tenant-id"),
	// }

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
				mockDriver := NewAzureDriver(mockPluginSPIImpl)

				// call setup before the create machine
				mockDriverClients, err := mockPluginSPIImpl.Setup(machineRequest.Secret)

				// Define all the client expectations here and then proceed with the function call
				fakeClients := mockDriverClients.(*mock.AzureDriverClients)

				var (
					ctx               = context.Background()
					vmName            = strings.ToLower(machineRequest.Machine.Name)
					resourceGroupName = providerSpec.ResourceGroup
					vnetName          = providerSpec.SubnetInfo.VnetName
					// vnetResourceGroup = resourceGroupName
					subnetName = providerSpec.SubnetInfo.SubnetName
					// nicName           = dependencyNameFromVMName(vmName, nicSuffix)
					// diskName          = dependencyNameFromVMName(vmName, diskSuffix)
					// vmImageRef        *compute.VirtualMachineImage
				)

				subnet := UnmarshalSubnet([]byte("{\"id\":\"/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/virtualNetworks/shoot--i538135--seed-az/subnets/shoot--i538135--seed-az-nodes\",\"name\":\"shoot--i538135--seed-az-nodes\",\"properties\":{\"addressPrefix\":\"10.250.0.0/16\",\"networkSecurityGroup\":{\"id\":\"/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/networkSecurityGroups/shoot--i538135--seed-az-workers\"},\"routeTable\":{\"id\":\"/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/routeTables/worker_route_table\"},\"serviceEndpoints\":[],\"ipConfigurations\":[{\"id\":\"/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-7jgvm-nic/ipConfigurations/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-7jgvm-nic\"},{\"id\":\"/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-rgqc2-nic/ipConfigurations/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-rgqc2-nic\"},{\"id\":\"/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-pfkg4-nic/ipConfigurations/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-pfkg4-nic\"}],\"delegations\":[],\"provisioningState\":\"Succeeded\",\"privateEndpointNetworkPolicies\":\"Enabled\",\"privateLinkServiceNetworkPolicies\":\"Enabled\"}}"))

				nicFuture := UnmarshalNICFuture([]byte("{\"method\":\"PUT\",\"pollingMethod\":\"RequestURI\",\"pollingURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/test-machine-deployment-oot-748df-95bhn-nic?api-version=2020-04-01\",\"lroState\":\"Succeeded\",\"resultURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/test-machine-deployment-oot-748df-95bhn-nic?api-version=2020-04-01\"}"))

				fakeClients.Subnet.EXPECT().Get(context.Background(),
					resourceGroupName,
					vnetName,
					subnetName,
					"").Return(subnet, nil)

				mockDriver.AzureProviderSpec = UnmarshalProviderSpec(mock.AzureProviderSpec)
				NICParameters := mockDriver.getNICParameters(vmName, &subnet)
				fakeClients.NIC.EXPECT().CreateOrUpdate(ctx, resourceGroupName, *NICParameters.Name, NICParameters).Return(nicFuture, nil)

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
				UnmarshalProviderSpec(mock.AzureProviderSpec),
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

// UnmarshalSubnet converts byte JSON to Subnet Struct
func UnmarshalSubnet(bytesSubnet []byte) network.Subnet {
	var subnet network.Subnet
	_ = json.Unmarshal(bytesSubnet, &subnet)
	// if err != nil {
	// 	return nil
	// }
	return subnet
}

func UnmarshalNICFuture(bytesNICFuture []byte) network.InterfacesCreateOrUpdateFuture {
	var nicFuture network.InterfacesCreateOrUpdateFuture
	_ = json.Unmarshal(bytesNICFuture, &nicFuture)
	return nicFuture
}

// UnmarshalProviderSpec converts byte JSON to AzureProviderSpec Struct
func UnmarshalProviderSpec(bytesProviderSpec []byte) *apis.AzureProviderSpec {
	var providerSpec apis.AzureProviderSpec
	err := json.Unmarshal(bytesProviderSpec, &providerSpec)
	if err != nil {
		return nil
	}
	return &providerSpec
}

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
