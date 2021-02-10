/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package azure

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"context"
	"encoding/json"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"k8s.io/apimachinery/pkg/runtime"

	apis "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	mock "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/mock"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	gomock "github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getStringPointer(s string) *string {
	return &s
}

func getBoolPointer(b bool) *bool {
	return &b
}

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
				mockDriver.Secret = machineRequest.Secret

				// call setup before the create machine
				mockDriverClients, err := mockPluginSPIImpl.Setup(machineRequest.Secret)

				// Define all the client expectations here and then proceed with the function call
				fakeClients := mockDriverClients.(*mock.AzureDriverClients)

				var (
					ctx               = context.Background()
					vmName            = strings.ToLower(machineRequest.Machine.Name)
					resourceGroupName = providerSpec.ResourceGroup
					vnetName          = providerSpec.SubnetInfo.VnetName
					subnetName        = providerSpec.SubnetInfo.SubnetName
					vmImageRef        *compute.VirtualMachineImage
				)

				subnet := network.Subnet{
					ID: getStringPointer("/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/" +
						"providers/Microsoft.Network/virtualNetworks/shoot--i538135--seed-az/subnets/shoot--i538135--seed-az-nodes"),
					Name: getStringPointer("shoot--i538135--seed-az-nodes"),
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: getStringPointer("10.250.0.0/16"),
						NetworkSecurityGroup: &network.SecurityGroup{
							ID: getStringPointer("/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/" +
								"shoot--i538135--seed-az/providers/Microsoft.Network/networkSecurityGroups/" +
								"shoot--i538135--seed-az-workers"),
						},
						RouteTable: &network.RouteTable{
							ID: getStringPointer("/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/" +
								"shoot--i538135--seed-az/providers/Microsoft.Network/routeTables/worker_route_table"),
						},
						IPConfigurations: &[]network.IPConfiguration{
							{
								ID: getStringPointer("/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/" +
									"shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/" +
									"shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-7jgvm-nic/ipConfigurations/" +
									"shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-7jgvm-nic"),
							},
							{
								ID: getStringPointer("/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/" +
									"shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/" +
									"shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-rgqc2-nic/ipConfigurations/" +
									"shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-rgqc2-nic"),
							},
							{
								ID: getStringPointer("/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/" +
									"shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/" +
									"shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-pfkg4-nic/ipConfigurations/" +
									"shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-pfkg4-nic"),
							},
						},
						ProvisioningState:                 "Succeeded",
						PrivateEndpointNetworkPolicies:    getStringPointer("Enabled"),
						PrivateLinkServiceNetworkPolicies: getStringPointer("Enabled"),
					},
				}

				NICFuture := UnmarshalNICFuture([]byte("{\"method\":\"PUT\",\"pollingMethod\":\"AsyncOperation\"," +
					"\"pollingURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/providers/Microsoft.Network/locations/westeurope/operations/e4469621-a170-4744-9aed-132d2992b230?api-version=2020-07-01\",\"lroState\":\"Succeeded\",\"resultURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Network/networkInterfaces/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-qtjm8-nic?api-version=2020-07-01\"}"))

				VMFutureAPI := UnmarshalVMCreateFuture([]byte("{\"method\":\"PUT\",\"pollingMethod\":\"AsyncOperation\",\"pollingURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/providers/Microsoft.Compute/locations/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succeeded\",\"resultURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Compute/virtualMachines/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-nnjnn?api-version=2020-06-01\"}"))

				splits := strings.Split(*providerSpec.Properties.StorageProfile.ImageReference.URN, ":")
				publisher := splits[0]
				offer := splits[1]
				sku := splits[2]
				version := splits[3]
				imageRef := &compute.ImageReference{
					Publisher: &publisher,
					Offer:     &offer,
					Sku:       &sku,
					Version:   &version,
				}

				fakeClients.Subnet.EXPECT().Get(ctx, resourceGroupName, vnetName, subnetName, "").Return(subnet, nil)

				mockDriver.AzureProviderSpec = &mock.AzureProviderSpec

				NICParameters := mockDriver.getNICParameters(vmName, &subnet)
				fakeClients.NIC.EXPECT().CreateOrUpdate(ctx, resourceGroupName, *NICParameters.Name, NICParameters).Return(NICFuture, nil)

				NICId := "/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/" +
					"Microsoft.Network/networkInterfaces/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-vs2lt-nic"

				vmImageRef = &compute.VirtualMachineImage{
					Name: mockDriver.AzureProviderSpec.Properties.StorageProfile.ImageReference.URN,
					VirtualMachineImageProperties: &compute.VirtualMachineImageProperties{
						Plan: nil,
					},
				}

				fakeClients.Images.EXPECT().Get(
					ctx,
					mockDriver.AzureProviderSpec.Location,
					*imageRef.Publisher,
					*imageRef.Offer,
					*imageRef.Sku,
					*imageRef.Version,
				).Return(*vmImageRef, nil)

				VMParameters := mockDriver.getVMParameters(vmName, vmImageRef, NICId)
				fakeClients.VM.EXPECT().CreateOrUpdate(ctx, resourceGroupName, *VMParameters.Name, VMParameters).Return(VMFutureAPI, nil)

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
				&mock.AzureProviderSpec,
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
			Entry("#2 Create machine with absence of client id in secret",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecretWithoutazureClientID),
				},
				nil,
				true,
				"machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating"+
					" ProviderSpec [secret azureClientId or clientID is required field]]]",
			),
		)
	})

	Describe("#Delete Machine", func() {

		DescribeTable("##Table",
			func(
				providerSpec *apis.AzureProviderSpec,
				machineRequest *driver.DeleteMachineRequest,
				machineResponse *driver.DeleteMachineResponse,
				errToHaveOccurred bool,
				errMessage string,
			) {

				// Create the mock controller and the mock clients
				controller := gomock.NewController(GinkgoT())
				mockPluginSPIImpl := mock.NewMockPluginSPIImpl(controller)
				mockDriver := NewAzureDriver(mockPluginSPIImpl)
				mockDriver.Secret = machineRequest.Secret

				// call setup before the create machine
				mockDriverClients, err := mockPluginSPIImpl.Setup(machineRequest.Secret)

				// Define all the client expectations here and then proceed with the function call
				fakeClients := mockDriverClients.(*mock.AzureDriverClients)

				var (
					ctx               = context.Background()
					resourceGroupName = providerSpec.ResourceGroup
				)

				mockDriver.AzureProviderSpec = providerSpec
				fakeClients.Group.EXPECT().Get(ctx, resourceGroupName).Return(resources.Group{}, nil)

				fakeClients.VM.EXPECT().Get(ctx, resourceGroupName, machineRequest.Machine.Name, compute.InstanceViewTypes("")).Return(compute.VirtualMachine{
					Name: getStringPointer(machineRequest.Machine.Name),
					VirtualMachineProperties: &compute.VirtualMachineProperties{
						StorageProfile: &compute.StorageProfile{
							DataDisks: &[]compute.DataDisk{},
						},
					},
				}, nil)

				VMFutureAPI := UnmarshalVMDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation\",\"pollingURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/providers/Microsoft.Compute/locations/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succeeded\",\"resultURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Compute/virtualMachines/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-nnjnn?api-version=2020-06-01\"}"))

				fakeClients.VM.EXPECT().Delete(ctx, resourceGroupName, machineRequest.Machine.Name, getBoolPointer(false)).Return(VMFutureAPI, nil)

				fakeClients.Disk.EXPECT().Get(ctx, resourceGroupName, machineRequest.Machine.Name+"-os-disk").Return(compute.Disk{
					ManagedBy: getStringPointer(""),
				}, nil)

				DisksFutureAPI := UnmarshalDisksDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation\",\"pollingURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/providers/Microsoft.Compute/locations/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succeeded\",\"resultURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Compute/virtualMachines/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-nnjnn?api-version=2020-06-01\"}"))

				fakeClients.Disk.EXPECT().Delete(ctx, resourceGroupName, machineRequest.Machine.Name+"-os-disk").Return(DisksFutureAPI, nil)

				fakeClients.NIC.EXPECT().Get(ctx, resourceGroupName, machineRequest.Machine.Name+"-nic", "").Return(network.Interface{
					InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
						VirtualMachine: nil,
					},
				}, nil)

				InterfacesFutureAPI := UnmarshalInterfacesDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation\",\"pollingURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/providers/Microsoft.Compute/locations/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succeeded\",\"resultURI\":\"https://management.azure.com/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/providers/Microsoft.Compute/virtualMachines/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-nnjnn?api-version=2020-06-01\"}"))

				fakeClients.NIC.EXPECT().Delete(ctx, resourceGroupName, machineRequest.Machine.Name+"-nic").Return(InterfacesFutureAPI, nil)

				// if there is no variation in the machine class (various scenarios) call the
				// machineRequest.MachineClass = newAzureMachineClass(providerSpec)
				response, err := mockDriver.DeleteMachine(ctx, machineRequest)

				if errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(machineResponse.LastKnownState).To(Equal(response.LastKnownState))
				}
			},

			Entry("#1 Delete a machine",
				&mock.AzureProviderSpec,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.DeleteMachineResponse{
					LastKnownState: "",
				},
				false,
				"",
			),
		)
	})
})

func UnmarshalNICFuture(bytesNICFuture []byte) network.InterfacesCreateOrUpdateFuture {
	var nicFuture network.InterfacesCreateOrUpdateFuture
	var futureAPI azure.Future

	_ = json.Unmarshal(bytesNICFuture, &futureAPI)
	nicFuture.FutureAPI = &futureAPI

	nicFuture.Result = func(nic network.InterfacesClient) (network.Interface, error) {
		return network.Interface{
			ID: getStringPointer("/subscriptions/00d2caa5-cd29-46f7-845a-2f8ee0360ef5/resourceGroups/shoot--i538135--seed-az/" +
				"providers/Microsoft.Network/networkInterfaces/shoot--i538135--seed-az-worker-m0exd-z2-b5bdd-vs2lt-nic"),
			Location: getStringPointer("westeurope"),
		}, nil
	}
	return nicFuture
}

func UnmarshalVMCreateFuture(bytesVMFutureAPI []byte) compute.VirtualMachinesCreateOrUpdateFuture {
	var VMFuture compute.VirtualMachinesCreateOrUpdateFuture
	var futureAPI azure.Future

	_ = json.Unmarshal(bytesVMFutureAPI, &futureAPI)
	VMFuture.FutureAPI = &futureAPI
	VMFuture.Result = func(vm compute.VirtualMachinesClient) (compute.VirtualMachine, error) {
		location := "westeurope"
		name := "dummy-machine"
		return compute.VirtualMachine{
			Location: &location,
			Name:     &name,
		}, nil
	}
	return VMFuture
}

func UnmarshalVMDeleteFuture(bytesVMFutureAPI []byte) compute.VirtualMachinesDeleteFuture {
	var VMFuture compute.VirtualMachinesDeleteFuture
	var futureAPI azure.Future

	_ = json.Unmarshal(bytesVMFutureAPI, &futureAPI)
	VMFuture.FutureAPI = &futureAPI
	VMFuture.Result = func(vm compute.VirtualMachinesClient) (autorest.Response, error) {
		return autorest.Response{
			Response: &http.Response{
				Status: "200 OK",
			},
		}, nil
	}
	return VMFuture
}

func UnmarshalDisksDeleteFuture(bytesDiskFutureAPI []byte) compute.DisksDeleteFuture {
	var DisksFuture compute.DisksDeleteFuture
	var futureAPI azure.Future

	_ = json.Unmarshal(bytesDiskFutureAPI, &futureAPI)
	DisksFuture.FutureAPI = &futureAPI
	DisksFuture.Result = func(compute.DisksClient) (autorest.Response, error) {
		return autorest.Response{
			Response: &http.Response{
				Status: "200 OK",
			},
		}, nil
	}
	return DisksFuture
}

func UnmarshalInterfacesDeleteFuture(bytesUnterfacesFutureAPI []byte) network.InterfacesDeleteFuture {
	var InterfacesFuture network.InterfacesDeleteFuture
	var futureAPI azure.Future

	_ = json.Unmarshal(bytesUnterfacesFutureAPI, &futureAPI)
	InterfacesFuture.FutureAPI = &futureAPI
	InterfacesFuture.Result = func(network.InterfacesClient) (autorest.Response, error) {
		return autorest.Response{
			Response: &http.Response{
				Status: "200 OK",
			},
		}, nil
	}
	return InterfacesFuture
}

func newMachine(name string) *v1alpha1.Machine {
	return &v1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func newAzureMachineClass(azureProviderSpec apis.AzureProviderSpec) *v1alpha1.MachineClass {
	byteData, _ := json.Marshal(azureProviderSpec)
	return &v1alpha1.MachineClass{
		ProviderSpec: runtime.RawExtension{
			Raw: byteData,
		},
	}
}

func newSecret(azureProviderSecretRaw map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		Data: azureProviderSecretRaw,
	}
}
