/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package azure contains the cloud provider specific implementations to manage machines
package azure

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
)

var _ = Describe("Utils", func() {

	Describe("generateDummyPublicKey", func() {

		It("should properly generate PublicKey string", func() {
			publicKey, err := generateDummyPublicKey()
			Expect(err).NotTo(HaveOccurred())

			Expect(publicKey).NotTo(Equal(""))
		})
	})

	Describe("getVMParameters", func() {
		var vmName string
		var networkInterfaceReferenceID string
		var providerSpec *api.AzureProviderSpec
		var azureProviderSecret *corev1.Secret

		BeforeEach(func() {
			vmName = "testName"
			networkInterfaceReferenceID = "testID"
			azureProviderSecret = &corev1.Secret{
				Data: map[string][]byte{
					"userData":            []byte("dummy-data"),
					"azureClientId":       []byte("dummy-client-id"),
					"azureClientSecret":   []byte("dummy-client-secret"),
					"azureSubscriptionId": []byte("dummy-subcription-id"),
					"azureTenantId":       []byte("dummy-tenant-id"),
				},
			}
			providerSpec = &api.AzureProviderSpec{
				Location: "westeurope",
				Properties: api.AzureVirtualMachineProperties{
					HardwareProfile: api.AzureHardwareProfile{
						VMSize: "Standard_DS2_v2",
					},
					StorageProfile: api.AzureStorageProfile{
						ImageReference: api.AzureImageReference{
							URN: pointer.String("sap:gardenlinux:greatest:27.1.0"),
						},
						OsDisk: api.AzureOSDisk{
							Caching: "None",
							ManagedDisk: api.AzureManagedDiskParameters{
								StorageAccountType: "Standard_LRS",
							},
							DiskSizeGB:   50,
							CreateOption: "FromImage",
						},
						DataDisks: []api.AzureDataDisk{
							{
								StorageAccountType: "Standard_LRS",
								Lun:                getInt32Pointer(1),
								DiskSizeGB:         50,
							},
						},
					},
					OsProfile: api.AzureOSProfile{
						AdminUsername: "core",
						LinuxConfiguration: api.AzureLinuxConfiguration{
							DisablePasswordAuthentication: true,
							SSH: api.AzureSSHConfiguration{
								PublicKeys: api.AzureSSHPublicKey{
									Path:    "/home/core/.ssh/authorized_keys",
									KeyData: "dummy keyData",
								},
							},
						},
					},
					Zone: pointer.Int(2),
				},
				ResourceGroup: "shoot--project--seed-az",
				SubnetInfo: api.AzureSubnetInfo{
					VnetName:   "shoot--project--seed-az",
					SubnetName: "shoot--project--seed-az-nodes",
				},
			}
		})

		It("should properly generate PublicKey when missing", func() {
			providerSpec.Properties.OsProfile.LinuxConfiguration.SSH.PublicKeys.KeyData = ""
			VMParameters, err := getVMParameters(vmName, nil, networkInterfaceReferenceID, providerSpec, azureProviderSecret)

			Expect(err).NotTo(HaveOccurred())
			Expect((*VMParameters.VirtualMachineProperties.OsProfile.LinuxConfiguration.SSH.PublicKeys)[0].KeyData).NotTo(Equal(""))
		})
	})
})
