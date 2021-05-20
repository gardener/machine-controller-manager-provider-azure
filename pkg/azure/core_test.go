/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package azure contains the cloud provider specific implementations to manage machines
package azure

import (
	"fmt"
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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getStringPointer(s string) *string {
	return &s
}

func getBoolPointer(b bool) *bool {
	return &b
}

func getInt32Pointer(i int32) *int32 {
	return &i
}

func getIntPointer(i int) *int {
	return &i
}

var (
	clusterTag = "kubernetes.io-cluster-shoot--project"
	roleTag    = "kubernetes.io-role-mcm"

	tags = map[string]*string{
		clusterTag: getStringPointer("yes"),
		roleTag:    getStringPointer("1"),
	}

	internalErrorPrefix = "machine codes error: code = [Internal] message = [machine codes error: code = [Internal] message = [error while validat" +
		"ing ProviderSpec [%s]]]"

	invalidArgumentErrorPrefix = "machine codes error: code = [InvalidArgument] message = [machine codes error: code = [Internal] message = [error while validat" +
		"ing ProviderSpec [%s]]]"

	secretError = "machine codes error: code = [Internal] message = [machine codes error: code = [Internal] message = [error while validat" +
		"ing ProviderSpec [secret %s or %s is required field]]]"

	providerSpecError = "machine codes error: code = [Internal] message = [machine codes error: code = [Internal] message = [error while v" +
		"alidating ProviderSpec [%s is required field]]]"

	providerSpecSubnetInfoError = "machine codes error: code = [Internal] message = [machine codes error: code = [Internal] message = [err" +
		"or while validating ProviderSpec [%s is a required subnet info]]]"

	machineClassProviderError = "machine codes error: code = [InvalidArgument] message = [requested for Provider '%s', we only support '" +
		ProviderAzure + "']"
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
				getSubnetError *autorest.DetailedError,
				nicCreateOrUpdateError *autorest.DetailedError,
				nicGetError *autorest.DetailedError,
				vmCreateOrUpdateError *autorest.DetailedError,
				errToHaveOccurred bool,
				errMessage string,
			) {
				var ctx = context.Background()

				// Create the mock controller and the mock clients
				controller := gomock.NewController(GinkgoT())
				mockPluginSPIImpl := mock.NewMockPluginSPIImpl(controller)
				mockDriver := NewAzureDriver(mockPluginSPIImpl)

				// call setup before the create machine
				mockDriverClients, err := mockPluginSPIImpl.Setup(machineRequest.Secret)
				Expect(err).ToNot(HaveOccurred())

				// Define all the client expectations here and then proceed with the function call
				fakeClients := mockDriverClients.(*mock.AzureDriverClients)

				assertNetworkResourcesForMachineCreation(mockDriver, fakeClients, providerSpec, machineRequest, getSubnetError, nicGetError, nicCreateOrUpdateError)
				assertVMResourcesForMachineCreation(mockDriver, fakeClients, providerSpec, machineRequest, vmCreateOrUpdateError)
				assertDiskResourcesForMachineCreation(mockDriver, fakeClients, providerSpec, machineRequest)

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
				nil,
				nil,
				nil,
				nil,
				false,
				"",
			),
			Entry("#1 Create a simple machine with wrong provider value in Machine Class",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClassWithProvider(mock.AzureProviderSpec, "aws"),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.CreateMachineResponse{
					ProviderID: "azure:///westeurope/dummy-machine",
					NodeName:   "dummy-machine",
				},
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(machineClassProviderError, "aws").Error(),
			),
			Entry("#2 Create machine without client id in secret",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecretWithoutazureClientID),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(secretError, "azureClientId", "clientID").Error(),
			),
			Entry("#3 Create machine without client secret in secret",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecretWithoutazureClientSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(secretError, "azureClientSecret", "clientSecret").Error(),
			),
			Entry("#4 Create machine without Tenant ID in secret",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecretWithoutazureTenantID),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(secretError, "azureTenantId", "tenantID").Error(),
			),
			Entry("#5 Create machine without Subscription ID in secret",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecretWithoutazureSubscriptionID),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(secretError, "azureSubscriptionId", "subscriptionID").Error(),
			),
			Entry("#6 Create machine without UserData in secret",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecretWithoutUserData),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(providerSpecError, "secret UserData").Error(),
			),
			Entry("#7 Create machine without location in providerSpec",
				&mock.AzureProviderSpecWithoutLocation,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutLocation),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(providerSpecError, "Region").Error(),
			),
			Entry("#8 Create machine without resource group in providerSpec",
				&mock.AzureProviderSpecWithoutResourceGroup,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutResourceGroup),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(providerSpecError, "ResourceGroup").Error(),
			),
			Entry("#9 Create machine without Vnet Name in providerSpec",
				&mock.AzureProviderSpecWithoutVnetName,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutVnetName),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(providerSpecSubnetInfoError, "VnetName").Error(),
			),
			Entry("#10 Create machine without Subnet Name in providerSpec",
				&mock.AzureProviderSpecWithoutSubnetName,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutSubnetName),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(providerSpecSubnetInfoError, "SubnetName").Error(),
			),
			Entry("#11 Create machine without VMSize in providerSpec",
				&mock.AzureProviderSpecWithoutVMSize,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutVMSize),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(providerSpecError, "VMSize").Error(),
			),
			Entry("#12 Create machine with improper ImageURN in providerSpec",
				&mock.AzureProviderSpecWithImproperImageURN,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithImproperImageURN),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.storageProfile.imageReference.urn: Required value: Invalid urn format, empty field").Error(),
			),
			Entry("#13 Create machine with negtive OS Disk Size in providerSpec",
				&mock.AzureProviderSpecWithNegativeOSDiskSize,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithNegativeOSDiskSize),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.storageProfile.osDisk.diskSizeGB: Required value: OSDisk size must be positive").Error(),
			),
			Entry("#14 Create machine without OS Disk Creation Option in providerSpec",
				&mock.AzureProviderSpecWithoutOSDiskCreateOption,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutOSDiskCreateOption),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.storageProfile.osDisk.createOption: Required value: OSDisk create option is required").Error(),
			),
			Entry("#15 Create machine without Admin Username in providerSpec",
				&mock.AzureProviderSpecWithoutAdminUserName,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutAdminUserName),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.osProfile.adminUsername: Required value: AdminUsername is required").Error(),
			),
			Entry("#16 Create machine with negative data disk size in providerSpec",
				&mock.AzureProviderSpecWithNegativeDataDiskSize,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithNegativeDataDiskSize),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.storageProfile.dataDisks[0].diskSizeGB: Required value: DataDisk size must be positive").Error(),
			),
			Entry("#17 Create machine without LUN in providerSpec",
				&mock.AzureProviderSpecWithoutLUN,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutLUN),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.storageProfile.dataDisks[0].lun: Required value: DataDisk Lun is required").Error(),
			),
			Entry("#18 Create machine with improper LUN in providerSpec",
				&mock.AzureProviderSpecWithImproperLUN,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithImproperLUN),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, fmt.Errorf("properties.storageProfile.dataDisks[0].lun: Invalid value: %d: must be between 0 and "+
					"63, inclusive", *mock.AzureProviderSpecWithImproperLUN.Properties.StorageProfile.DataDisks[0].Lun).Error()).Error(),
			),
			Entry("#19 Create machine without Storage Account Type in providerSpec",
				&mock.AzureProviderSpecWithoutDiskStorageAccountType,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutDiskStorageAccountType),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.storageProfile.dataDisks[0].storageAccountType: Required value: DataDisk storage account type is required").Error(),
			),
			Entry("#20 Create machine with duplicated LUN in providerSpec",
				&mock.AzureProviderSpecWithDuplicatedLUN,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithDuplicatedLUN),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, fmt.Errorf("properties.storageProfile.dataDisks: Invalid value: 1: Data Disk Lun '%d' duplicated 2 times, Lun must be unique", *mock.AzureProviderSpecWithDuplicatedLUN.Properties.StorageProfile.DataDisks[0].Lun).Error()).Error(),
			),
			Entry("#21 Create machine without Machineset, Zone & availability set in providerSpec",
				&mock.AzureProviderSpecWithoutZMA,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutZMA),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.zone|.machineSet|.availabilitySet: Forbidden: Machine need to be assigned to a zone, a MachineSet or an AvailabilitySet").Error(),
			),
			Entry("#22 Create machine with Machineset, Zone & availability set in providerSpec",
				&mock.AzureProviderSpecWithZMA,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithZMA),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.zone|.machineSet|.availabilitySet: Forbidden: Machine cannot be assigned to a zone, a MachineSet and an AvailabilitySet in parallel").Error(),
			),
			Entry("#23 Create machine with only Machineset & availability set in providerSpec",
				&mock.AzureProviderSpecWithMAOnly,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithMAOnly),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.machineSet|.availabilitySet: Forbidden: Machine cannot be assigned a MachineSet and an AvailabilitySet in parallel").Error(),
			),
			Entry("#24 Create machine with invalid machine set in providerSpec",
				&mock.AzureProviderSpecWithInvalidMachineSet,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithInvalidMachineSet),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, fmt.Errorf("properties.machineSet: Invalid value: \"%s\": Invalid MachineSet kind. Use either '%s' or '%s'", mock.AzureProviderSpecWithInvalidMachineSet.Properties.MachineSet.Kind, apis.MachineSetKindVMO, apis.MachineSetKindAvailabilitySet).Error()).Error(),
			),
			Entry("#25 Create machine with empty cluster name in providerSpec",
				&mock.AzureProviderSpecWithEmptyClusterNameTag,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithEmptyClusterNameTag),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "providerSpec.kubernetes.io-cluster-: Required value: Tag required of the form kubernetes.io-cluster-****").Error(),
			),
			Entry("#26 Create machine with empty node role tag in providerSpec",
				&mock.AzureProviderSpecWithEmptyNodeRoleTag,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithEmptyNodeRoleTag),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "providerSpec.kubernetes.io-role-: Required value: Tag required of the form kubernetes.io-role-****").Error(),
			),
			Entry("#27 Create a simple machine with getSubnet Error",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				&autorest.DetailedError{
					StatusCode: 500,
					Message:    "Internal error while fetching the Subnet data",
					Response: &http.Response{
						Status:     "Internal",
						StatusCode: 500,
					},
				},
				nil,
				nil,
				nil,
				true,
				"machine codes error: code = [Internal] message = [#: Internal error while fetching the Subnet data: StatusCode=500]",
			),
			Entry("#28 Create a simple machine with nicCreateOrUpdate Error",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				&autorest.DetailedError{
					StatusCode: 500,
					Message:    "Internal error while Creating/Updating NIC",
					Response: &http.Response{
						Status:     "Internal",
						StatusCode: 500,
					},
				},
				nil,
				nil,
				true,
				"machine codes error: code = [Internal] message = [#: Internal error while Creating/Updating NIC: StatusCode=500]",
			),
			Entry("#29 Create a simple machine with unmarshalling error on providerSpec",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClassWithError(),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				true,
				"machine codes error: code = [Internal] message = [machine codes error: code = [Internal] message = [invalid character '\"' after object key]]",
			),
			Entry("#30 Create a simple machine with vmCreateOrUpdate Error",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				nil,
				&autorest.DetailedError{
					StatusCode: 500,
					Message:    "Internal error while Creating/Updating VM",
					Response: &http.Response{
						Status:     "Internal",
						StatusCode: 500,
					},
				},
				true,
				"machine codes error: code = [Internal] message = [#: Internal error while Creating/Updating VM: StatusCode=500]",
			),
			Entry("#31 Create a simple machine with nicGet() Internal Error",
				&mock.AzureProviderSpec,
				&driver.CreateMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				nil,
				&autorest.DetailedError{
					StatusCode: 500,
					Message:    "Internal error while fetching NIC details",
					Response: &http.Response{
						Status:     "Internal",
						StatusCode: 500,
					},
				},
				nil,
				true,
				"machine codes error: code = [Internal] message = [#: Internal error while fetching NIC details: StatusCode=500]",
			),
		)
	})

	Describe("#Delete Machine", func() {

		DescribeTable("##Table",
			func(
				providerSpec *apis.AzureProviderSpec,
				machineRequest *driver.DeleteMachineRequest,
				machineResponse *driver.DeleteMachineResponse,
				getVMError *autorest.DetailedError,
				attachedNIC bool,
				attachedOSDisk bool,
				attachedDataDisk bool,
				getGroupError *autorest.DetailedError,
				errToHaveOccurred bool,
				errMessage string,
			) {

				// Create the mock controller and the mock clients
				controller := gomock.NewController(GinkgoT())
				mockPluginSPIImpl := mock.NewMockPluginSPIImpl(controller)
				mockDriver := NewAzureDriver(mockPluginSPIImpl)

				// call setup before the create machine
				mockDriverClients, err := mockPluginSPIImpl.Setup(machineRequest.Secret)
				Expect(err).ToNot(HaveOccurred())

				// Define all the client expectations here and then proceed with the function call
				fakeClients := mockDriverClients.(*mock.AzureDriverClients)

				var (
					ctx               = context.Background()
					resourceGroupName = providerSpec.ResourceGroup
				)

				if getGroupError != nil {
					fakeClients.Group.EXPECT().Get(gomock.Any(), resourceGroupName).Return(resources.Group{}, *getGroupError)
				} else {
					fakeClients.Group.EXPECT().Get(gomock.Any(), resourceGroupName).Return(resources.Group{}, nil)
				}

				assertVMResourcesForMachineDeletion(mockDriver, fakeClients, providerSpec, machineRequest, getVMError)
				assertDiskResourcesForMachineDeletion(mockDriver, fakeClients, providerSpec, machineRequest, attachedOSDisk, attachedDataDisk)
				assertNetworkResourcesForMachineDeletion(mockDriver, fakeClients, providerSpec, machineRequest, attachedNIC)

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
				nil,
				false,
				false,
				false,
				nil,
				false,
				nil,
			),
			Entry("#2 Delete a machine",
				&mock.AzureProviderSpec,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClassWithProvider(mock.AzureProviderSpec, "aws"),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.DeleteMachineResponse{
					LastKnownState: "",
				},
				nil,
				false,
				false,
				false,
				nil,
				true,
				fmt.Errorf(machineClassProviderError, "aws").Error(),
			),
			Entry("#2 Delete a machine while a NIC is still attached",
				&mock.AzureProviderSpec,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.DeleteMachineResponse{
					LastKnownState: "",
				},
				nil,
				true,
				false,
				false,
				nil,
				true,
				"machine codes error: code = [Internal] message = [cannot delete NIC dummy-machine-nic because it is attached to VM dummy-"+
					"machine-id]",
			),

			Entry("#3 Delete a machine while an OS disk is still attached",
				&mock.AzureProviderSpec,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.DeleteMachineResponse{
					LastKnownState: "",
				},
				nil,
				false,
				true,
				false,
				nil,
				true,
				"machine codes error: code = [Internal] message = [cannot delete disk dummy-machine-os-disk because it is attached to VM d"+
					"ummy-machine-id]",
			),

			Entry("#4 Delete a machine while a Data disk is still attached",
				&mock.AzureProviderSpecWithDataDisks,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithDataDisks),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.DeleteMachineResponse{
					LastKnownState: "",
				},
				nil,
				false,
				false,
				true,
				nil,
				true,
				"machine codes error: code = [Internal] message = [cannot delete disk dummy-machine-"+
					fmt.Sprintf("%d", *mock.AzureProviderSpecWithDataDisks.Properties.StorageProfile.DataDisks[0].Lun)+"-data-disk because"+
					" it is attached to VM dummy-machine-id]",
			),

			Entry("#5 Delete a machine while a Data disk with Name is still attached",
				&mock.AzureProviderSpecWithDataDisksWithName,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithDataDisksWithName),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.DeleteMachineResponse{
					LastKnownState: "",
				},
				nil,
				false,
				false,
				true,
				nil,
				true,
				"machine codes error: code = [Internal] message = [cannot delete disk dummy-machine-"+
					fmt.Sprintf("%s-%d", mock.AzureProviderSpecWithDataDisksWithName.Properties.StorageProfile.DataDisks[0].Name,
						*mock.AzureProviderSpecWithDataDisksWithName.Properties.StorageProfile.DataDisks[0].Lun)+
					"-data-disk because it is attached to VM dummy-machine-id]",
			),

			Entry("#6 Delete a machine without admin username in the providerSpec",
				&mock.AzureProviderSpecWithoutAdminUserName,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithoutAdminUserName),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				false,
				false,
				false,
				nil,
				true,
				fmt.Errorf(internalErrorPrefix, "properties.osProfile.adminUsername: Required value: AdminUsername is required").Error(),
			),
			Entry("#7 Delete a machine where group does not exist",
				&mock.AzureProviderSpec,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				false,
				false,
				false,
				&autorest.DetailedError{
					StatusCode: 404,
					Message:    "Resource Group not found",
					Response: &http.Response{
						Status:     "NotFound",
						StatusCode: 404,
					},
				},
				true,
				"machine codes error: code = [NotFound] message = [#: Resource Group not found: StatusCode=404]",
			),
			Entry("#8 Delete a machine where group returns an Internal error",
				&mock.AzureProviderSpec,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				nil,
				false,
				false,
				false,
				&autorest.DetailedError{
					StatusCode: 500,
					Message:    "Internal error with Resource group",
					Response: &http.Response{
						Status:     "Internal",
						StatusCode: 500,
					},
				},
				true,
				"machine codes error: code = [Internal] message = [#: Internal error with Resource group: StatusCode=500]",
			),
			Entry("#9 Delete a machine where getVM has internal error",
				&mock.AzureProviderSpec,
				&driver.DeleteMachineRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				&autorest.DetailedError{
					StatusCode: 500,
					Message:    "Internal error VM resource",
					Response: &http.Response{
						Status:     "Internal",
						StatusCode: 500,
					},
				},
				false,
				false,
				false,
				nil,
				true,
				"machine codes error: code = [Internal] message = [#: Internal error VM resource: StatusCode=500]",
			),
		)
	})

	Describe("#List Machines", func() {

		DescribeTable("##Table",
			func(
				providerSpec *apis.AzureProviderSpec,
				machineRequest *driver.ListMachinesRequest,
				machineResponse *driver.ListMachinesResponse,
				nextWithContextError bool,
				vmListError *autorest.DetailedError,
				errToHaveOccurred bool,
				errMessage string,
			) {
				// Create the mock controlelr and mock clients
				controller := gomock.NewController(GinkgoT())
				mockPluginSPIImpl := mock.NewMockPluginSPIImpl(controller)
				mockDriver := NewAzureDriver(mockPluginSPIImpl)

				// call setup before the create machine
				mockDriverClients, err := mockPluginSPIImpl.Setup(machineRequest.Secret)
				Expect(err).ToNot(HaveOccurred())

				// Define all the client expectations here and then proceed with the function call
				fakeClients := mockDriverClients.(*mock.AzureDriverClients)

				var (
					ctx               = context.Background()
					resourceGroupName = providerSpec.ResourceGroup
				)

				assertVMResourcesForListingMachine(mockDriver, fakeClients, resourceGroupName, nextWithContextError, vmListError)
				response, err := mockDriver.ListMachines(ctx, machineRequest)

				if errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(response.MachineList["azure:///westeurope/dummy-machine"]).To(Equal("dummy-machine"))
				}
			},

			Entry("#1 List machines",
				&mock.AzureProviderSpec,
				&driver.ListMachinesRequest{
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.ListMachinesResponse{
					MachineList: map[string]string{
						"azure:///westeurope/dummy-machine": "dummy-machine",
					},
				},
				false,
				nil,
				false,
				"",
			),
			Entry("#2 List machines with wrong MachineClass Provider",
				&mock.AzureProviderSpec,
				&driver.ListMachinesRequest{
					MachineClass: newAzureMachineClassWithProvider(mock.AzureProviderSpec, "aws"),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.ListMachinesResponse{
					MachineList: map[string]string{
						"azure:///westeurope/dummy-machine": "dummy-machine",
					},
				},
				false,
				nil,
				true,
				fmt.Errorf(machineClassProviderError, "aws").Error(),
			),
			Entry("#3 List machines with VM List error scenario",
				&mock.AzureProviderSpec,
				&driver.ListMachinesRequest{
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				false,
				&autorest.DetailedError{
					StatusCode: 500,
					Message:    "Internal resource group",
					Response: &http.Response{
						Status:     "Internal",
						StatusCode: 500,
					},
				},
				true,
				"machine codes error: code = [Internal] message = [#: Internal resource group: StatusCode=500]",
			),
			Entry("#4 List machines with VM List error scenario",
				&mock.AzureProviderSpec,
				&driver.ListMachinesRequest{
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				true,
				nil,
				true,
				"machine codes error: code = [Internal] message = [Error fetching the next Virtual Machine in the page]",
			),
			Entry("#5 List machines with wrong MachineClass",
				&mock.AzureProviderSpec,
				&driver.ListMachinesRequest{
					MachineClass: newAzureMachineClass(mock.AzureProviderSpecWithNegativeDataDiskSize),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.ListMachinesResponse{
					MachineList: map[string]string{
						"azure:///westeurope/dummy-machine": "dummy-machine",
					},
				},
				false,
				nil,
				true,
				fmt.Errorf(invalidArgumentErrorPrefix, "properties.storageProfile.dataDisks[0].diskSizeGB: Required value:"+
					" DataDisk size must be positive").Error(),
			),
		)
	})

	Describe("#GetMachinesStatus", func() {

		DescribeTable("##Table",
			func(
				providerSpec *apis.AzureProviderSpec,
				machineRequest *driver.GetMachineStatusRequest,
				machineResponse *driver.GetMachineStatusResponse,
				vmlr compute.VirtualMachineListResult,
				vmListError *autorest.DetailedError,
				errToHaveOccurred bool,
				errMessage string,
			) {
				// Create the mock controlelr and mock clients
				controller := gomock.NewController(GinkgoT())
				mockPluginSPIImpl := mock.NewMockPluginSPIImpl(controller)
				mockDriver := NewAzureDriver(mockPluginSPIImpl)

				// call setup before the create machine
				mockDriverClients, err := mockPluginSPIImpl.Setup(machineRequest.Secret)
				Expect(err).ToNot(HaveOccurred())

				// Define all the client expectations here and then proceed with the function call
				fakeClients := mockDriverClients.(*mock.AzureDriverClients)

				var (
					ctx               = context.Background()
					resourceGroupName = providerSpec.ResourceGroup
				)

				vmlrp := compute.NewVirtualMachineListResultPage(
					vmlr,
					func(context.Context, compute.VirtualMachineListResult) (compute.VirtualMachineListResult, error) {
						return compute.VirtualMachineListResult{}, nil
					},
				)

				if vmListError != nil {
					fakeClients.VM.EXPECT().List(gomock.Any(), resourceGroupName).Return(
						compute.VirtualMachineListResultPage{}, vmListError,
					)
				} else {
					fakeClients.VM.EXPECT().List(gomock.Any(), resourceGroupName).Return(
						vmlrp, nil,
					)
				}

				fakeClients.NIC.EXPECT().List(gomock.Any(), resourceGroupName).Return(network.InterfaceListResultPage{}, nil)
				fakeClients.Disk.EXPECT().ListByResourceGroup(gomock.Any(), resourceGroupName).Return(compute.DiskListPage{}, nil)

				response, err := mockDriver.GetMachineStatus(ctx, machineRequest)

				if errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(response.ProviderID).To(Equal("azure:///westeurope/dummy-machine"))
				}
			},

			Entry("#1 GetMachineStatus of valid machine",
				&mock.AzureProviderSpec,
				&driver.GetMachineStatusRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.GetMachineStatusResponse{
					NodeName:   "dummy-machine",
					ProviderID: "azure:///westeurope/dummy-machine",
				},
				compute.VirtualMachineListResult{
					Value: &[]compute.VirtualMachine{
						{
							Name:     getStringPointer("dummy-machine"),
							Location: getStringPointer("westeurope"),
							Tags: map[string]*string{
								"kubernetes.io-cluster-shoot--project--seed-az": getStringPointer("yes"),
								"kubernetes.io-role-mcm":                        getStringPointer("1"),
							},
						},
					},
					NextLink: getStringPointer(""),
				},
				nil,
				false,
				"",
			),
			Entry("#1 GetMachineStatus of machine with wrong MachineClass reference",
				&mock.AzureProviderSpec,
				&driver.GetMachineStatusRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClassWithProvider(mock.AzureProviderSpec, "aws"),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.GetMachineStatusResponse{
					NodeName:   "dummy-machine",
					ProviderID: "azure:///westeurope/dummy-machine",
				},
				compute.VirtualMachineListResult{
					Value: &[]compute.VirtualMachine{
						{
							Name:     getStringPointer("dummy-machine"),
							Location: getStringPointer("westeurope"),
						},
					},
					NextLink: getStringPointer(""),
				},
				nil,
				true,
				fmt.Errorf(machineClassProviderError, "aws").Error(),
			),
			Entry("#2 GetMachineStatus of non existing machine",
				&mock.AzureProviderSpec,
				&driver.GetMachineStatusRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				nil,
				compute.VirtualMachineListResult{
					Value: &[]compute.VirtualMachine{
						{
							Name:     getStringPointer("dummy-machine-1"),
							Location: getStringPointer("westeurope"),
						},
					},
					NextLink: getStringPointer(""),
				},
				nil,
				true,
				"machine codes error: code = [NotFound] message = [machine 'dummy-machine' not found]",
			),
			Entry("#3 GetMachineStatus of machine with error while listing the machines",
				&mock.AzureProviderSpec,
				&driver.GetMachineStatusRequest{
					Machine:      newMachine("dummy-machine"),
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					Secret:       newSecret(azureProviderSecret),
				},
				&driver.GetMachineStatusResponse{
					NodeName:   "dummy-machine",
					ProviderID: "azure:///westeurope/dummy-machine",
				},
				compute.VirtualMachineListResult{
					Value: &[]compute.VirtualMachine{
						{
							Name:     getStringPointer("dummy-machine"),
							Location: getStringPointer("westeurope"),
						},
					},
					NextLink: getStringPointer(""),
				},
				&autorest.DetailedError{
					StatusCode: 500,
					Message:    "Internal error while listing machines",
					Response: &http.Response{
						Status:     "Internal",
						StatusCode: 500,
					},
				},
				true,
				"machine codes error: code = [Internal] message = [#: Internal error while listing machines: StatusCode=500]",
			),
		)
	})

	Describe("#GetVolumeIDs", func() {

		var hostPathPVSpec = &corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/mnt/data",
				},
			},
		}

		DescribeTable("##Table",
			func(
				machineRequest *driver.GetVolumeIDsRequest,
				machineResponse *driver.GetVolumeIDsResponse,
				errToHaveOccurred bool,
				errMessage string,
			) {
				// Create the mock controlelr and mock clients
				controller := gomock.NewController(GinkgoT())
				mockPluginSPIImpl := mock.NewMockPluginSPIImpl(controller)
				mockDriver := NewAzureDriver(mockPluginSPIImpl)

				var (
					ctx = context.Background()
				)

				response, err := mockDriver.GetVolumeIDs(ctx, machineRequest)

				if errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(response.VolumeIDs).To(Equal(machineResponse.VolumeIDs))
				}
			},

			Entry("#1 Get Volume IDs of AzureDisks",
				&driver.GetVolumeIDsRequest{
					PVSpecs: []*corev1.PersistentVolumeSpec{
						{
							StorageClassName: "example",
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								AzureDisk: &corev1.AzureDiskVolumeSource{
									DiskName: "example-disk",
								},
							},
						},
						hostPathPVSpec,
					},
				},
				&driver.GetVolumeIDsResponse{
					VolumeIDs: []string{
						"example-disk",
					},
				},
				false,
				"",
			),
			Entry("#2 Get Volume IDs without any AzureDisks",
				&driver.GetVolumeIDsRequest{
					PVSpecs: []*corev1.PersistentVolumeSpec{
						{
							StorageClassName: "example",
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								AzureDisk: nil,
							},
						},
					},
				},
				&driver.GetVolumeIDsResponse{
					VolumeIDs: []string{},
				},
				false,
				"",
			),
			Entry("#3 Get Volume IDs with CSI Azure out-of-tree PV (with .spec.csi.volumeHandle)",
				&driver.GetVolumeIDsRequest{
					PVSpecs: []*corev1.PersistentVolumeSpec{
						{
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								CSI: &corev1.CSIPersistentVolumeSource{
									Driver:       "disk.csi.azure.com",
									VolumeHandle: "vol-1",
								},
							},
						},
						{
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								CSI: &corev1.CSIPersistentVolumeSource{
									Driver:       "io.kubernetes.storage.mock",
									VolumeHandle: "vol-2",
								},
							},
						},
						hostPathPVSpec,
					},
				},
				&driver.GetVolumeIDsResponse{
					VolumeIDs: []string{
						"vol-1",
					},
				},
				false,
				"",
			),
		)
	})

	Describe("#GenerateMachineClassForMigration", func() {

		DescribeTable("##Table",
			func(
				machineRequest *driver.GenerateMachineClassForMigrationRequest,
				machineResponse *driver.GenerateMachineClassForMigrationResponse,
				errToHaveOccurred bool,
				errMessage string,
			) {
				// Create the mock controlelr and mock clients
				controller := gomock.NewController(GinkgoT())
				mockPluginSPIImpl := mock.NewMockPluginSPIImpl(controller)
				mockDriver := NewAzureDriver(mockPluginSPIImpl)

				var (
					ctx = context.Background()
				)

				_, err := mockDriver.GenerateMachineClassForMigration(ctx, machineRequest)

				response := getMigratedMachineClass(machineRequest.ProviderSpecificMachineClass)

				if errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(machineRequest.MachineClass).To(Equal(response))
				}
			},

			Entry("#1 Generate machine class for migration",
				&driver.GenerateMachineClassForMigrationRequest{
					ProviderSpecificMachineClass: &v1alpha1.AzureMachineClass{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test-azure",
							Namespace: "default",
							Labels: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
							Annotations: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
							Finalizers: []string{
								"mcm/finalizer",
							},
						},
						TypeMeta: v1.TypeMeta{},
						Spec: v1alpha1.AzureMachineClassSpec{
							Location:      "westeurope",
							ResourceGroup: "sample-resource-group",
							SubnetInfo: v1alpha1.AzureSubnetInfo{
								VnetName:   "sample-vnet",
								SubnetName: "sample-subnet",
							},
							SecretRef: &corev1.SecretReference{
								Name:      "test-secret",
								Namespace: "default",
							},
							Tags: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
							Properties: v1alpha1.AzureVirtualMachineProperties{
								HardwareProfile: v1alpha1.AzureHardwareProfile{
									VMSize: "sample-vmsize",
								},
								NetworkProfile: v1alpha1.AzureNetworkProfile{
									AcceleratedNetworking: getBoolPointer(false),
								},
								StorageProfile: v1alpha1.AzureStorageProfile{
									ImageReference: v1alpha1.AzureImageReference{
										URN: getStringPointer("sample-urn"),
									},
									OsDisk: v1alpha1.AzureOSDisk{
										Caching:      "None",
										DiskSizeGB:   50,
										CreateOption: "FromImage",
									},
									DataDisks: []v1alpha1.AzureDataDisk{
										{
											Name:               "sdb",
											Lun:                getInt32Pointer(0),
											Caching:            "None",
											StorageAccountType: "Standard_LRS",
										},
										{
											Name:               "sdb",
											Lun:                getInt32Pointer(1),
											Caching:            "None",
											StorageAccountType: "Standard_LRS",
										},
									},
								},
								OsProfile: v1alpha1.AzureOSProfile{
									AdminUsername: "admin-name",
									LinuxConfiguration: v1alpha1.AzureLinuxConfiguration{
										DisablePasswordAuthentication: true,
										SSH: v1alpha1.AzureSSHConfiguration{
											PublicKeys: v1alpha1.AzureSSHPublicKey{
												Path:    "/path/to/public-key/in/machine",
												KeyData: "public-key-data",
											},
										},
									},
								},
								IdentityID: getStringPointer("/subscriptions/subscription-id/resourceGroups/resource-group-name/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name"),
								Zone:       getIntPointer(1),
								MachineSet: &v1alpha1.AzureMachineSetConfig{
									ID:   "/subscriptions/subscription-id/resourceGroups/resource-group-name/providers/Microsoft.Compute/azureMachineSetResourceType/machine-set-name",
									Kind: "availabilityset",
								},
								AvailabilitySet: &v1alpha1.AzureSubResource{
									ID: "/subscriptions/subscription-id/resourceGroups/resource-group-name/providers/Microsoft.Compute/availabilitySets/availablity-set-name",
								},
							},
						},
					},
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					ClassSpec: &v1alpha1.ClassSpec{
						Kind: AzureMachineClassKind,
						Name: "test-azure",
					},
				},
				&driver.GenerateMachineClassForMigrationResponse{},
				false,
				"",
			),
			Entry("#2 Generate machine class for migration for invalid machine class kind",
				&driver.GenerateMachineClassForMigrationRequest{
					ProviderSpecificMachineClass: &v1alpha1.AzureMachineClass{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test-azure",
							Namespace: "default",
							Labels: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
							Annotations: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
							Finalizers: []string{
								"mcm/finalizer",
							},
						},
						TypeMeta: v1.TypeMeta{},
						Spec: v1alpha1.AzureMachineClassSpec{
							Location:      "westeurope",
							ResourceGroup: "sample-resource-group",
							SubnetInfo: v1alpha1.AzureSubnetInfo{
								VnetName:   "sample-vnet",
								SubnetName: "sample-subnet",
							},
							SecretRef: &corev1.SecretReference{
								Name:      "test-secret",
								Namespace: "default",
							},
							Tags: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
							Properties: v1alpha1.AzureVirtualMachineProperties{
								HardwareProfile: v1alpha1.AzureHardwareProfile{
									VMSize: "sample-vmsize",
								},
								NetworkProfile: v1alpha1.AzureNetworkProfile{
									AcceleratedNetworking: getBoolPointer(false),
								},
								StorageProfile: v1alpha1.AzureStorageProfile{
									ImageReference: v1alpha1.AzureImageReference{
										URN: getStringPointer("sample-urn"),
									},
									OsDisk: v1alpha1.AzureOSDisk{
										Caching:      "None",
										DiskSizeGB:   50,
										CreateOption: "FromImage",
									},
									DataDisks: []v1alpha1.AzureDataDisk{
										{
											Name:               "sdb",
											Lun:                getInt32Pointer(0),
											Caching:            "None",
											StorageAccountType: "Standard_LRS",
										},
										{
											Name:               "sdb",
											Lun:                getInt32Pointer(1),
											Caching:            "None",
											StorageAccountType: "Standard_LRS",
										},
									},
								},
								OsProfile: v1alpha1.AzureOSProfile{
									AdminUsername: "admin-name",
									LinuxConfiguration: v1alpha1.AzureLinuxConfiguration{
										DisablePasswordAuthentication: true,
										SSH: v1alpha1.AzureSSHConfiguration{
											PublicKeys: v1alpha1.AzureSSHPublicKey{
												Path:    "/path/to/public-key/in/machine",
												KeyData: "public-key-data",
											},
										},
									},
								},
								IdentityID: getStringPointer("/subscriptions/subscription-id/resourceGroups/resource-group-name/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name"),
								Zone:       getIntPointer(1),
								MachineSet: &v1alpha1.AzureMachineSetConfig{
									ID:   "/subscriptions/subscription-id/resourceGroups/resource-group-name/providers/Microsoft.Compute/azureMachineSetResourceType/machine-set-name",
									Kind: "availabilityset",
								},
								AvailabilitySet: &v1alpha1.AzureSubResource{
									ID: "/subscriptions/subscription-id/resourceGroups/resource-group-name/providers/Microsoft.Compute/availabilitySets/availablity-set-name",
								},
							},
						},
					},
					MachineClass: newAzureMachineClass(mock.AzureProviderSpec),
					ClassSpec: &v1alpha1.ClassSpec{
						Kind: "DummyMachineClassKind",
						Name: "test-azure",
					},
				},
				&driver.GenerateMachineClassForMigrationResponse{},
				true,
				"machine codes error: code = [Internal] message = [Migration cannot be done for this machineClass kind]",
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
			ID: getStringPointer("/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-group/" +
				"providers/Microsoft.Network/networkInterfaces/dummy-resource-group-worker-m0exd-z2-b5bdd-vs2lt-nic"),
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
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
	}
}

func newAzureMachineClassWithProvider(azureProviderSpec apis.AzureProviderSpec, provider string) *v1alpha1.MachineClass {
	byteData, _ := json.Marshal(azureProviderSpec)
	return &v1alpha1.MachineClass{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
		},
		ProviderSpec: runtime.RawExtension{
			Raw: byteData,
		},
		Provider: provider,
	}
}

func newAzureMachineClass(azureProviderSpec apis.AzureProviderSpec) *v1alpha1.MachineClass {
	byteData, _ := json.Marshal(azureProviderSpec)
	return &v1alpha1.MachineClass{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
		},
		ProviderSpec: runtime.RawExtension{
			Raw: byteData,
		},
		Provider: ProviderAzure,
	}
}

func newAzureMachineClassWithError() *v1alpha1.MachineClass {
	byteData := []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--project--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--project--seed-az-nodes\",\"vnetName\":\"shoot--project--seed-az\"},\"tags\":{\"Name\":\"shoot--project--seed-az\",\"kubernetes.io-cluster-shoot--project--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\"\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	return &v1alpha1.MachineClass{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
		},
		ProviderSpec: runtime.RawExtension{
			Raw: byteData,
		},
		Provider: ProviderAzure,
	}
}

func newSecret(azureProviderSecretRaw map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		Data: azureProviderSecretRaw,
	}
}

func getMigratedMachineClass(providerSpecificMachineClass interface{}) *v1alpha1.MachineClass {

	var (
		properties apis.AzureVirtualMachineProperties
		subnetInfo apis.AzureSubnetInfo
	)

	data, _ := json.Marshal(providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Spec.Properties)
	_ = json.Unmarshal(data, &properties)

	data, _ = json.Marshal(providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Spec.SubnetInfo)
	_ = json.Unmarshal(data, &subnetInfo)

	providerSpec := &apis.AzureProviderSpec{
		Location:      providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Spec.Location,
		Tags:          providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Spec.Tags,
		Properties:    properties,
		ResourceGroup: providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Spec.ResourceGroup,
		SubnetInfo:    subnetInfo,
	}

	// Marshal providerSpec into Raw Bytes
	providerSpecMarshal, _ := json.Marshal(providerSpec)

	machineClass := &v1alpha1.MachineClass{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:        providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Name,
			Namespace:   providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Namespace,
			Labels:      providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Labels,
			Annotations: providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Annotations,
			Finalizers:  providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Finalizers,
		},
		ProviderSpec: runtime.RawExtension{
			Raw: providerSpecMarshal,
		},
		Provider:  ProviderAzure,
		SecretRef: providerSpecificMachineClass.(*v1alpha1.AzureMachineClass).Spec.SecretRef,
	}

	return machineClass
}

func assertNetworkResourcesForMachineCreation(
	mockDriver *MachinePlugin,
	fakeClients *mock.AzureDriverClients,
	providerSpec *apis.AzureProviderSpec,
	machineRequest *driver.CreateMachineRequest,
	getSubnetError,
	nicGetError, nicCreateOrUpdateError *autorest.DetailedError,
) {

	var (
		vmName            = strings.ToLower(machineRequest.Machine.Name)
		resourceGroupName = providerSpec.ResourceGroup
		vnetName          = providerSpec.SubnetInfo.VnetName
		subnetName        = providerSpec.SubnetInfo.SubnetName
		nicName           = dependencyNameFromVMName(vmName, nicSuffix)
	)

	subnet := network.Subnet{
		ID: getStringPointer("/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-group/provide" +
			"rs/Microsoft.Network/virtualNetworks/dummy-resource-group/subnets/dummy-resource-group-nodes"),
		Name: getStringPointer("dummy-resource-group-nodes"),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: getStringPointer("10.250.0.0/16"),
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: getStringPointer("/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-group" +
					"/providers/Microsoft.Network/networkSecurityGroups/dummy-resource-group-workers"),
			},
			RouteTable: &network.RouteTable{
				ID: getStringPointer("/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-group" +
					"/providers/Microsoft.Network/routeTables/worker_route_table"),
			},
			IPConfigurations: &[]network.IPConfiguration{
				{
					ID: getStringPointer("/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-g" +
						"roup/providers/Microsoft.Network/networkInterfaces/dummy-resource-group-worker-m0exd-z2-b5bdd-7jgvm-" +
						"nic/ipConfigurations/dummy-resource-group-worker-m0exd-z2-b5bdd-7jgvm-nic"),
				},
				{
					ID: getStringPointer("/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-g" +
						"roup/providers/Microsoft.Network/networkInterfaces/dummy-resource-group-worker-m0exd-z2-b5bdd-rgqc2-" +
						"nic/ipConfigurations/dummy-resource-group-worker-m0exd-z2-b5bdd-rgqc2-nic"),
				},
				{
					ID: getStringPointer("/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-g" +
						"roup/providers/Microsoft.Network/networkInterfaces/dummy-resource-group-worker-m0exd-z2-b5bdd-pfkg4-" +
						"nic/ipConfigurations/dummy-resource-group-worker-m0exd-z2-b5bdd-pfkg4-nic"),
				},
			},
			ProvisioningState:                 "Succeeded",
			PrivateEndpointNetworkPolicies:    getStringPointer("Enabled"),
			PrivateLinkServiceNetworkPolicies: getStringPointer("Enabled"),
		},
	}

	NICFuture := UnmarshalNICFuture([]byte("{\"method\":\"PUT\",\"pollingMethod\":\"AsyncOperation\",\"pollingURI\":\"https:/" +
		"/management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/providers/Microsoft.Network/locations/weste" +
		"urope/operations/e4469621-a170-4744-9aed-132d2992b230?api-version=2020-07-01\",\"lroState\":\"Succeeded\",\"resultUR" +
		"I\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-" +
		"group/providers/Microsoft.Network/networkInterfaces/dummy-resource-group-worker-m0exd-z2-b5bdd-qtjm8-nic?api-version" +
		"=2020-07-01\"}"))

	if getSubnetError != nil {
		fakeClients.Subnet.EXPECT().Get(gomock.Any(), resourceGroupName, vnetName, subnetName, "").Return(subnet, *getSubnetError)
	} else {
		fakeClients.Subnet.EXPECT().Get(gomock.Any(), resourceGroupName, vnetName, subnetName, "").Return(subnet, nil)
	}

	NICParameters := mockDriver.getNICParameters(vmName, &subnet, providerSpec)

	if nicGetError != nil {
		fakeClients.NIC.EXPECT().Get(gomock.Any(), resourceGroupName, nicName, "").Return(network.Interface{}, *nicGetError)
	} else {
		fakeClients.NIC.EXPECT().Get(gomock.Any(), resourceGroupName, nicName, "").Return(network.Interface{}, autorest.DetailedError{
			Response: &http.Response{
				StatusCode: 404,
			},
			StatusCode: 404,
		})
	}

	if nicCreateOrUpdateError != nil {
		fakeClients.NIC.EXPECT().CreateOrUpdate(gomock.Any(), resourceGroupName, *NICParameters.Name,
			NICParameters).Return(network.InterfacesCreateOrUpdateFuture{}, *nicCreateOrUpdateError)

	} else {
		fakeClients.NIC.EXPECT().CreateOrUpdate(gomock.Any(), resourceGroupName, *NICParameters.Name,
			NICParameters).Return(NICFuture, nil)
	}

	InterfacesFutureAPI := UnmarshalInterfacesDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation" +
		"\",\"pollingURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/providers/Micros" +
		"oft.Compute/locations/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState" +
		"\":\"Succeeded\",\"resultURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/res" +
		"ourceGroups/dummy-resource-group/providers/Microsoft.Compute/virtualMachines/dummy-resource-group-worker-m0exd-z2-b5" +
		"bdd-nnjnn?api-version=2020-06-01\"}"))

	fakeClients.NIC.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-nic").Return(InterfacesFutureAPI, nil)

}

func assertVMResourcesForMachineCreation(
	mockDriver *MachinePlugin,
	fakeClients *mock.AzureDriverClients,
	providerSpec *apis.AzureProviderSpec,
	machineRequest *driver.CreateMachineRequest,
	vmCreateOrUpdateError *autorest.DetailedError,
) {

	var (
		vmName            = strings.ToLower(machineRequest.Machine.Name)
		resourceGroupName = providerSpec.ResourceGroup
		nicName           = dependencyNameFromVMName(vmName, nicSuffix)
		vmImageRef        *compute.VirtualMachineImage
	)
	VMFutureAPI := UnmarshalVMCreateFuture([]byte("{\"method\":\"PUT\",\"pollingMethod\":\"AsyncOperation\",\"pollingURI\":\"" +
		"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/providers/Microsoft.Compute/location" +
		"s/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succeeded\",\"r" +
		"esultURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-re" +
		"source-group/providers/Microsoft.Compute/virtualMachines/dummy-resource-group-worker-m0exd-z2-b5bdd-nnjnn?api-versio" +
		"n=2020-06-01\"}"))

	// mocked methods for driver.deleteVMNICDisk
	fakeClients.VM.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name, compute.InstanceViewTypes("")).Return(compute.VirtualMachine{
		Name: getStringPointer(machineRequest.Machine.Name),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			StorageProfile: &compute.StorageProfile{
				DataDisks: &[]compute.DataDisk{},
			},
		},
	}, nil)

	VMDeleteFutureAPI := UnmarshalVMDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation\",\"pollingURI\"" +
		":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/providers/Microsoft.Compute/locat" +
		"ions/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succeeded\"," +
		"\"resultURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy" +
		"-resource-group/providers/Microsoft.Compute/virtualMachines/dummy-resource-group-worker-m0exd-z2-b5bdd-nnjnn?api-ver" +
		"sion=2020-06-01\"}"))

	fakeClients.VM.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name, getBoolPointer(false)).Return(VMDeleteFutureAPI, nil)

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

	vmImageRef = &compute.VirtualMachineImage{
		Name: providerSpec.Properties.StorageProfile.ImageReference.URN,
		VirtualMachineImageProperties: &compute.VirtualMachineImageProperties{
			Plan: nil,
		},
	}

	fakeClients.Images.EXPECT().Get(
		gomock.Any(),
		providerSpec.Location,
		*imageRef.Publisher,
		*imageRef.Offer,
		*imageRef.Sku,
		*imageRef.Version,
	).Return(*vmImageRef, nil)

	NICId := "/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-group/providers/Microsoft.Net" +
		"work/networkInterfaces/dummy-resource-group-worker-m0exd-z2-b5bdd-vs2lt-nic"

	if vmCreateOrUpdateError != nil {

		fakeClients.NIC.EXPECT().Get(gomock.Any(), resourceGroupName, nicName, "").Return(network.Interface{}, autorest.DetailedError{
			Response: &http.Response{
				StatusCode: 404,
			},
			StatusCode: 404,
		})
		VMParameters := mockDriver.getVMParameters(vmName, vmImageRef, NICId, providerSpec, machineRequest.Secret)
		fakeClients.VM.EXPECT().CreateOrUpdate(gomock.Any(), resourceGroupName, *VMParameters.Name, VMParameters).Return(compute.VirtualMachinesCreateOrUpdateFuture{}, *vmCreateOrUpdateError)
	} else {

		fakeClients.NIC.EXPECT().Get(gomock.Any(), resourceGroupName, nicName, "").Return(network.Interface{}, autorest.DetailedError{
			Response: &http.Response{
				StatusCode: 404,
			},
			StatusCode: 404,
		})
		VMParameters := mockDriver.getVMParameters(vmName, vmImageRef, NICId, providerSpec, machineRequest.Secret)
		fakeClients.VM.EXPECT().CreateOrUpdate(gomock.Any(), resourceGroupName, *VMParameters.Name, VMParameters).Return(VMFutureAPI, nil)
	}
}

func assertDiskResourcesForMachineCreation(
	mockDriver *MachinePlugin,
	fakeClients *mock.AzureDriverClients,
	providerSpec *apis.AzureProviderSpec,
	machineRequest *driver.CreateMachineRequest,
) {
	var resourceGroupName = providerSpec.ResourceGroup
	fakeClients.Disk.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-os-disk").Return(compute.Disk{
		ManagedBy: nil,
	}, nil)

	fakeClients.Disk.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"data-disk-1-data-disk").Return(compute.Disk{
		ManagedBy: nil,
	}, nil)

	fakeClients.Disk.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-1-data-disk").Return(compute.Disk{
		ManagedBy: nil,
	}, nil)

	DisksFutureAPI := UnmarshalDisksDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation\",\"pollin" +
		"gURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/providers/Microsoft.Compute" +
		"/locations/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succee" +
		"ded\",\"resultURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups" +
		"/dummy-resource-group/providers/Microsoft.Compute/virtualMachines/dummy-resource-group-worker-m0exd-z2-b5bdd-nnjnn?a" +
		"pi-version=2020-06-01\"}"))

	fakeClients.Disk.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-os-disk").Return(DisksFutureAPI, nil)

	fakeClients.Disk.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"data-disk-1-data-disk").Return(DisksFutureAPI, nil)

	fakeClients.Disk.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-1-data-disk").Return(DisksFutureAPI, nil)

}

func assertVMResourcesForMachineDeletion(
	mockDriver *MachinePlugin,
	fakeClients *mock.AzureDriverClients,
	providerSpec *apis.AzureProviderSpec,
	machineRequest *driver.DeleteMachineRequest,
	getVMError *autorest.DetailedError,
) {
	resourceGroupName := providerSpec.ResourceGroup
	if getVMError != nil {
		fakeClients.VM.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name, compute.InstanceViewTypes("")).Return(compute.VirtualMachine{}, *getVMError)
	} else {
		fakeClients.VM.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name, compute.InstanceViewTypes("")).Return(compute.VirtualMachine{
			Name: getStringPointer(machineRequest.Machine.Name),
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				StorageProfile: &compute.StorageProfile{
					DataDisks: &[]compute.DataDisk{},
				},
			},
		}, nil)
	}

	VMFutureAPI := UnmarshalVMDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation\",\"pollingURI\"" +
		":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/providers/Microsoft.Compute/locat" +
		"ions/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succeeded\"," +
		"\"resultURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy" +
		"-resource-group/providers/Microsoft.Compute/virtualMachines/dummy-resource-group-worker-m0exd-z2-b5bdd-nnjnn?api-ver" +
		"sion=2020-06-01\"}"))

	fakeClients.VM.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name, getBoolPointer(false)).Return(VMFutureAPI, nil)

}

func assertDiskResourcesForMachineDeletion(
	mockDriver *MachinePlugin,
	fakeClients *mock.AzureDriverClients,
	providerSpec *apis.AzureProviderSpec,
	machineRequest *driver.DeleteMachineRequest,
	attachedOSDisk, attachedDataDisk bool,
) {
	var (
		diskName          string
		resourceGroupName = providerSpec.ResourceGroup
	)
	if attachedOSDisk {
		fakeClients.Disk.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-os-disk").Return(compute.Disk{
			ManagedBy: getStringPointer("dummy-machine-id"),
		}, nil)
	} else {
		fakeClients.Disk.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-os-disk").Return(compute.Disk{
			ManagedBy: nil,
		}, nil)
	}

	if attachedDataDisk {
		if providerSpec.Properties.StorageProfile.DataDisks[0].Name != "" {
			diskName = fmt.Sprintf("%s-%d", providerSpec.Properties.StorageProfile.DataDisks[0].Name, *providerSpec.Properties.StorageProfile.DataDisks[0].Lun)
		} else {
			diskName = fmt.Sprintf("%d", *providerSpec.Properties.StorageProfile.DataDisks[0].Lun)
		}
		fakeClients.Disk.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-"+diskName+"-data-disk").Return(compute.Disk{
			ManagedBy: getStringPointer("dummy-machine-id"),
		}, nil)

	} else {
		if providerSpec.Properties.StorageProfile.DataDisks[0].Name != "" {
			diskName = fmt.Sprintf("%s-%d", providerSpec.Properties.StorageProfile.DataDisks[0].Name, *providerSpec.Properties.StorageProfile.DataDisks[0].Lun)
		} else {
			diskName = fmt.Sprintf("%d", *providerSpec.Properties.StorageProfile.DataDisks[0].Lun)
		}
		fakeClients.Disk.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-"+diskName+"-data-disk").Return(compute.Disk{
			ManagedBy: nil,
		}, nil)
	}

	DisksFutureAPI := UnmarshalDisksDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation\",\"pollin" +
		"gURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/providers/Microsoft.Compute" +
		"/locations/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState\":\"Succee" +
		"ded\",\"resultURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups" +
		"/dummy-resource-group/providers/Microsoft.Compute/virtualMachines/dummy-resource-group-worker-m0exd-z2-b5bdd-nnjnn?a" +
		"pi-version=2020-06-01\"}"))

	fakeClients.Disk.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-os-disk").Return(DisksFutureAPI, nil)
	fakeClients.Disk.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-1-data-disk").Return(DisksFutureAPI, nil)

}

func assertNetworkResourcesForMachineDeletion(

	mockDriver *MachinePlugin,
	fakeClients *mock.AzureDriverClients,
	providerSpec *apis.AzureProviderSpec,
	machineRequest *driver.DeleteMachineRequest,
	attachedNIC bool,
) {
	resourceGroupName := providerSpec.ResourceGroup

	NICId := "/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/resourceGroups/dummy-resource-group/providers/Microsoft.Net" +
		"work/networkInterfaces/dummy-resource-group-worker-m0exd-z2-b5bdd-vs2lt-nic"

	if attachedNIC {

		fakeClients.NIC.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-nic", "").Return(network.Interface{
			InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
				VirtualMachine: &network.SubResource{
					ID: getStringPointer("dummy-machine-id"),
				},
			},
		}, nil)

	} else {

		fakeClients.NIC.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-nic", "").Return(network.Interface{
			InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
				VirtualMachine: nil,
			},
			ID: &NICId,
		}, nil)
	}

	InterfacesFutureAPI := UnmarshalInterfacesDeleteFuture([]byte("{\"method\":\"DELETE\",\"pollingMethod\":\"AsyncOperation" +
		"\",\"pollingURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/providers/Micros" +
		"oft.Compute/locations/westeurope/operations/e4a4273e-f571-420f-9629-aa6b95d46e7c?api-version=2020-06-01\",\"lroState" +
		"\":\"Succeeded\",\"resultURI\":\"https://management.azure.com/subscriptions/c222a292-7836-42da-836e-984c6e269ef0/res" +
		"ourceGroups/dummy-resource-group/providers/Microsoft.Compute/virtualMachines/dummy-resource-group-worker-m0exd-z2-b5" +
		"bdd-nnjnn?api-version=2020-06-01\"}"))

	fakeClients.NIC.EXPECT().Delete(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-nic").Return(InterfacesFutureAPI, nil)

	fakeClients.NIC.EXPECT().Get(gomock.Any(), resourceGroupName, machineRequest.Machine.Name+"-nic", "").Return(network.Interface{}, autorest.DetailedError{
		StatusCode: 404,
		Response: &http.Response{
			StatusCode: 404,
		},
	})
}

func assertVMResourcesForListingMachine(
	mockDriver *MachinePlugin,
	fakeClients *mock.AzureDriverClients,
	resourceGroupName string,
	nextWithContextError bool,
	vmListError *autorest.DetailedError,
) {
	var vmlr compute.VirtualMachineListResultPage
	if !nextWithContextError {
		vmlr = compute.NewVirtualMachineListResultPage(
			compute.VirtualMachineListResult{
				Value: &[]compute.VirtualMachine{
					{
						Name:     getStringPointer("dummy-machine"),
						Location: getStringPointer("westeurope"),
						Tags:     tags,
					},
				},
				NextLink: getStringPointer(""),
			},
			func(context.Context, compute.VirtualMachineListResult) (compute.VirtualMachineListResult, error) {
				return compute.VirtualMachineListResult{}, nil
			},
		)
	} else {
		vmlr = compute.NewVirtualMachineListResultPage(
			compute.VirtualMachineListResult{
				Value: &[]compute.VirtualMachine{
					{
						Name:     getStringPointer("dummy-machine"),
						Location: getStringPointer("westeurope"),
						Tags:     tags,
					},
				},
				NextLink: getStringPointer(""),
			},
			func(context.Context, compute.VirtualMachineListResult) (compute.VirtualMachineListResult, error) {
				return compute.VirtualMachineListResult{}, fmt.Errorf("Error fetching the next Virtual Machine in the page")
			},
		)
	}

	if vmListError != nil {
		fakeClients.VM.EXPECT().List(gomock.Any(), resourceGroupName).Return(
			compute.VirtualMachineListResultPage{}, *vmListError,
		)
	} else {
		fakeClients.VM.EXPECT().List(gomock.Any(), resourceGroupName).Return(
			vmlr, nil,
		)
	}

	fakeClients.NIC.EXPECT().List(gomock.Any(), resourceGroupName).Return(network.InterfaceListResultPage{}, nil)
	fakeClients.Disk.EXPECT().ListByResourceGroup(gomock.Any(), resourceGroupName).Return(compute.DiskListPage{}, nil)

}
