/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"context"

	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	fake "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/mock"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
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
		"azureClientID":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	azureProviderSecretWithoutazureClientID := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientID":       []byte(""),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	azureProviderSecretWithoutazureClientSecret := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientID":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte(""),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	azureProviderSecretWithoutazureTenantID := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientID":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte(""),
	}

	azureProviderSecretWithoutazureSubscriptionID := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientID":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte(""),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	azureProviderSecretWithoutUserData := map[string][]byte{
		"userData":            []byte(""),
		"azureClientID":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
		"azureTenantId":       []byte("dummy-tenant-id"),
	}

	Describe("#Create Machine", func() {

		type setup struct {
		}

		type action struct {
			machineRequest *driver.CreateMachineRequest
		}

		type expect struct {
			machineResponse   *driver.CreateMachineResponse
			errToHaveOccurred bool
			errMessage        string
		}

		type data struct {
			setup  setup
			action action
			expect expect
		}

		DescribeTable("##Table",
			func(data *data) {

				var mockPluginSPIImpl *fake.PluginSPIImpl

				mockPluginSPIImpl = &fake.PluginSPIImpl{}
				ms := fake.NewFakeAzureDriver(mockPluginSPIImpl)

				ctx := context.Background()
				response, err := ms.CreateMachine(ctx, data.action.machineRequest)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(data.expect.errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(data.expect.machineResponse.ProviderID).To(Equal(response.ProviderID))
					Expect(data.expect.machineResponse.NodeName).To(Equal(response.NodeName))
				}
			},

			Entry("#1 Create a simple machine", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "azure:///westeurope/dummy-machine",
						NodeName:   "dummy-machine",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("#2 CreateMachine fails: Absence of UserData in secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecretWithoutUserData),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Secret UserData is required field]]]",
				},
			}),
			Entry("#3 CreateMachine fails: Absence of Location in providerspec", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutLocation),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Region is required field]]]",
				},
			}),
			Entry("#4 CreateMachine fails: Unmarshalling for provider spec fails empty providerSpec", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass([]byte("")),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [unexpected end of JSON input]]",
				},
			}),
			Entry("#5 CreateMachine fails: Absence of Resource Group in providerSpec", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutResourceGroup),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Resource Group Name is required field]]]",
				},
			}),

			Entry("#6 CreateMachine fails: Absence of VnetName in providerSpec.subnetinfo", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutVnetName),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [VnetName is required for the subnet info]]]",
				},
			}),
			Entry("#7 CreateMachine fails: Absence of SubnetName in providerSpec.subnetinfo", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutSubnetName),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Subnet name is required for subnet info]]]",
				},
			}),
			Entry("#8 CreateMachine fails: Absence of VMSize in providerSpec.properties.HardwareProfile", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutVMSize),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [VMSize is required]]]",
				},
			}),
			Entry("#10 CreateMachine fails: Absence of Image URN", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutImageURN),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.storageProfile.imageReference: Required value: must specify either a image id or an urn]]]",
				},
			}),
			Entry("#11 CreateMachine fails: Improper of Image URN", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithImproperImageURN),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.storageProfile.imageReference.urn: Required value: Invalid urn format]]]",
				},
			}),
			Entry("#12 CreateMachine fails: Improper of Image URN with empty fields", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithEmptyFieldImageURN),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.storageProfile.imageReference.urn: Required value: Invalid urn format, empty field]]]",
				},
			}),
			Entry("#13 CreateMachine fails: Negative OS disk size", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithNegativeOSDiskSize),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.storageProfile.osDisk.diskSizeGB: Required value: OSDisk size must be positive]]]",
				},
			}),
			Entry("#14 CreateMachine fails: Absence of OSDisk Create Option", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutOSDiskCreateOption),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.storageProfile.osDisk.createOption: Required value: OSDisk create option is required]]]",
				},
			}),
			Entry("#15 CreateMachine fails: Absence of AdminUserName in OSProfile", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutAdminUserName),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.osProfile.adminUsername: Required value: AdminUsername is required]]]",
				},
			}),
			Entry("#16 CreateMachine fails: Absence of Zone, MachineSet and AvailabilitySet", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithoutZMA),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.zone|.machineSet|.availabilitySet: Forbidden: Machine need to be assigned to a zone, a MachineSet or an AvailabilitySet]]]",
				},
			}),
			Entry("#17 CreateMachine fails: Presence of Zone, MachineSet and AvailablitySet together", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithZMA),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.zone|.machineSet|.availabilitySet: Forbidden: Machine cannot be assigned to a zone, a MachineSet and an AvailabilitySet in parallel]]]",
				},
			}),
			Entry("#18 CreateMachine fails: Assigning MachineSet and AvailablitySet together with no zone", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithMAOnly),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.machineSet|.availabilitySet: Forbidden: Machine cannot be assigned a MachineSet and an AvailabilitySet in parallel]]]",
				},
			}),
			Entry("#19 CreateMachine fails: Invalid MachineSetKind", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithInvalidMachineSet),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [properties.machineSet: Invalid value: \"machinekind\": Invalid MachineSet kind. Use either '" + api.MachineSetKindVMO + "' or '" + api.MachineSetKindAvailabilitySet + "']]]",
				},
			}),
			Entry("#20 CreateMachine fails: Empty clusterName Tag", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithEmptyClusterNameTag),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [providerSpec.kubernetes.io-cluster-: Required value: Tag required of the form kubernetes.io-cluster-****]]]",
				},
			}),
			Entry("#21 CreateMachine fails: Empty nodeRole Tag", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpecWithEmptyNodeRoleTag),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [providerSpec.kubernetes.io-role-: Required value: Tag required of the form kubernetes.io-role-****]]]",
				},
			}),
			Entry("#22 CreateMachine fails: Absence of azureClientID in secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecretWithoutazureClientID),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Secret azureClientID is required field]]]",
				},
			}),
			Entry("#23 CreateMachine fails: Absence of azureClientSecret in secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecretWithoutazureClientSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Secret azureClientSecret is required field]]]",
				},
			}),
			Entry("#24 CreateMachine fails: Absence of azureTenantId in secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecretWithoutazureTenantID),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Secret azureTenantId is required field]]]",
				},
			}),
			Entry("#25 CreateMachine fails: Absence of azureSubscriptionId in secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecretWithoutazureSubscriptionID),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Unknown] message = [machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Secret azureSubscriptionId is required field]]]",
				},
			}),
		)
	})

	Describe("#Delete Machine", func() {

		type setup struct {
		}

		type action struct {
			machineRequest *driver.DeleteMachineRequest
		}

		type expect struct {
			machineResponse   *driver.DeleteMachineResponse
			errToHaveOccurred bool
			errMessage        string
		}

		type data struct {
			setup  setup
			action action
			expect expect
		}

		DescribeTable("##Table",
			func(data *data) {

				var mockPluginSPIImpl *fake.PluginSPIImpl

				mockPluginSPIImpl = &fake.PluginSPIImpl{}
				ms := fake.NewFakeAzureDriver(mockPluginSPIImpl)

				ctx := context.Background()
				response, err := ms.DeleteMachine(ctx, data.action.machineRequest)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(data.expect.errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(data.expect.machineResponse.LastKnownState).To(Equal(response.LastKnownState))
				}
			},

			Entry("#1 Delete a simple machine", &data{
				action: action{
					machineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					machineResponse:   &driver.DeleteMachineResponse{},
					errToHaveOccurred: false,
				},
			}),
		)
	})

	Describe("#Get Machine Status", func() {

		type setup struct {
		}

		type action struct {
			machineRequest *driver.GetMachineStatusRequest
		}

		type expect struct {
			machineResponse   *driver.GetMachineStatusResponse
			errToHaveOccurred bool
			errMessage        string
		}

		type data struct {
			setup  setup
			action action
			expect expect
		}

		DescribeTable("# Get Machine Status Table",
			func(data *data) {

				var mockPluginSPIImpl *fake.PluginSPIImpl

				mockPluginSPIImpl = &fake.PluginSPIImpl{}
				ms := fake.NewFakeAzureDriver(mockPluginSPIImpl)

				ctx := context.Background()
				response, err := ms.GetMachineStatus(ctx, data.action.machineRequest)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(data.expect.errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(data.expect.machineResponse.NodeName).To(Equal(response.NodeName))
				}
			},

			Entry("#1 Get Machine Status a simple machine", &data{
				action: action{
					machineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					machineResponse: &driver.GetMachineStatusResponse{
						NodeName: "dummy",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("#2 Get Machine Status of a non existing machine", &data{
				action: action{
					machineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(fake.AzureProviderSpec),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errMessage:        "machine codes error: code = [NotFound] message = [Machine 'dummy-machine' not found]",
					errToHaveOccurred: true,
				},
			}),
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
