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

	fake "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/fake"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	//FailAtNotFound is the error message returned when a resource is not found
	FailAtNotFound string = "machine codes error: code = [NotFound] message = [machine name=non-existent-dummy-machine, uuid= not found]"
	//FailAtJSONUnmarshalling is the error message returned when an malformed JSON is sent to the plugin by the caller
	FailAtJSONUnmarshalling string = "machine codes error: code = [Internal] message = [Machine status \"dummy-machine\" failed on decodeProviderSpecAndSecret: machine codes error: code = [Internal] message = [unexpected end of JSON input]]"
	//CreateFailAtJSONUnmarshalling is the error message returned when an malformed JSON is sent to the plugin by the caller
	CreateFailAtJSONUnmarshalling string = "machine codes error: code = [Internal] message = [Create machine \"dummy-machine\" failed on decodeProviderSpecAndSecret: machine codes error: code = [Internal] message = [unexpected end of JSON input]]"
	//DeleteFailAtJSONUnmarshalling is the error message returned when an malformed JSON is sent to the plugin by the caller
	DeleteFailAtJSONUnmarshalling string = "machine codes error: code = [Internal] message = [Delete machine \"dummy-machine\" failed on decodeProviderSpecAndSecret: machine codes error: code = [Internal] message = [unexpected end of JSON input]]"
	//ListFailAtJSONUnmarshalling is the error message returned when an malformed JSON is sent to the plugin by the caller
	ListFailAtJSONUnmarshalling string = "machine codes error: code = [Internal] message = [List machines failed on decodeProviderSpecAndSecret: machine codes error: code = [Internal] message = [unexpected end of JSON input]]"
	//FailAtNoSecretsPassed is the error message returned when no secrets are passed to the the plugin by the caller
	FailAtNoSecretsPassed string = "machine codes error: code = [Internal] message = [Create machine \"dummy-machine\" failed on decodeProviderSpecAndSecret: machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Secret serviceAccountJSON is required field Secret userData is required field]]]"
	//FailAtSecretsWithNoUserData is the error message returned when secrets map has no userdata provided by the caller
	FailAtSecretsWithNoUserData string = "machine codes error: code = [Internal] message = [Create machine \"dummy-machine\" failed on decodeProviderSpecAndSecret: machine codes error: code = [Internal] message = [Error while validating ProviderSpec [Secret userData is required field]]]"
	// FailAtInvalidProjectID is the error returned when an invalid project id value is provided by the caller
	FailAtInvalidProjectID string = "machine codes error: code = [Internal] message = [Create machine \"dummy-machine\" failed: json: cannot unmarshal number into Go struct field .project_id of type string]"
	//FailAtInvalidZonePostCall is the  error returned when a post call should fail with an invalid zone is sent in the POST call -- this is used to simulate server error
	FailAtInvalidZonePostCall string = "machine codes error: code = [Internal] message = [Create machine \"dummy-machine\" failed: googleapi: got HTTP response code 400 with body: Invalid post zone\n]"
	//FailAtInvalidZoneListCall is the  error returned when a list call should fail with an invalid zone is sent in the LIST call -- this is used to simulate server error
	FailAtInvalidZoneListCall string = "machine codes error: code = [Internal] message = [Machine status \"dummy-machine\" failed: googleapi: got HTTP response code 400 with body: Invalid list zone\n]"
	//CreateFailAtInvalidZoneListCall is the  error returned when a list call should fail with an invalid zone is sent in the CREATE call -- this is used to simulate server error
	CreateFailAtInvalidZoneListCall string = "machine codes error: code = [Internal] message = [Create machine \"dummy-machine\" failed: googleapi: got HTTP response code 400 with body: Invalid list zone\n]"
	//DeleteFailAtInvalidZoneListCall is the  error returned when a list call should fail with an invalid zone is sent in the DELETE call -- this is used to simulate server error
	DeleteFailAtInvalidZoneListCall string = "machine codes error: code = [Internal] message = [Delete machine \"dummy-machine\" failed: googleapi: got HTTP response code 400 with body: Invalid list zone\n]"
	//ListFailAtInvalidZoneListCall is the  error returned when a list call should fail with an invalid zone is sent in the LIST call -- this is used to simulate server error
	ListFailAtInvalidZoneListCall string = "machine codes error: code = [Internal] message = [List machines failed: googleapi: got HTTP response code 400 with body: Invalid list zone\n]"
	//FailAtMethodNotImplemented is the error returned for methods which are not yet implemented
	FailAtMethodNotImplemented string = "rpc error: code = Unimplemented desc = "
	FailAtSpecValidation       string = "machine codes error: code = [Internal] message = [Create machine \"dummy-machine\" failed on decodeProviderSpecAndSecret: machine codes error: code = [Internal] message = [Error while validating ProviderSpec [spec.zone: Required value: zone is required]]]"
	FailAtNonExistingMachine   string = "rpc error: code = NotFound desc = Machine with the name \"non-existent-dummy-machine\" not found"
)

var _ = Describe("MachineController", func() {

	// This is the value of ProviderSpec key of Kind Machine Class for Azure
	azureProviderSpec := []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// azureProviderSpecWithoutLocation := []byte("{\"location\":,\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	azureProviderSecret := map[string][]byte{
		"userData":            []byte("dummy-data"),
		"azureClientId":       []byte("dummy-client-id"),
		"azureClientSecret":   []byte("dummy-client-secret"),
		"azureSubscriptionId": []byte("dummy-subcription-id"),
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

			Entry("Create a simple machine", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(azureProviderSpec),
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
			Entry("CreateMachine fails: Absence of UserData in secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(azureProviderSpec),
						Secret:       newSecret(azureProviderSecretWithoutUserData),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = rpc error: code = Internal desc = Error while validating ProviderSpec [Secret UserData is required field]",
				},
			}),
			Entry("CreateMachine fails: Unmarshalling for provider spec fails", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine("dummy"),
						MachineClass: newAzureMachineClass([]byte("")),
						Secret:       newSecret(azureProviderSecret),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = rpc error: code = Internal desc = unexpected end of JSON input",
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

			Entry("Delete a simple machine", &data{
				action: action{
					machineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine("dummy-machine"),
						MachineClass: newAzureMachineClass(azureProviderSpec),
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
