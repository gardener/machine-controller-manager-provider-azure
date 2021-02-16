/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/
package mock

import api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"

func getUrn(urn string) *string {
	return &urn
}

func getZone(zone int) *int {
	return &zone
}

var (

	// AzureProviderSpec is the value of ProviderSpec key of Kind Machine Class for Azure
	AzureProviderSpec = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithoutLocation is the providerSpec without location value
	AzureProviderSpecWithoutLocation = api.AzureProviderSpec{
		Location: "",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}
	// AzureProviderSpecWithoutResourceGroup is providerSpec without resource group value
	AzureProviderSpecWithoutResourceGroup = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}
	// AzureProviderSpecWithoutVnetName is providerSpec without vNetName
	AzureProviderSpecWithoutVnetName = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithoutSubnetName Kind Machine Class for Azure
	AzureProviderSpecWithoutSubnetName = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithoutVMSize without VMSize
	AzureProviderSpecWithoutVMSize = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithoutImageURN without ImageURN
	AzureProviderSpecWithoutImageURN = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn(""),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithImproperImageURN with Improper ImageURN
	AzureProviderSpecWithImproperImageURN = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap::greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithEmptyFieldImageURN with EmptyField Image URN
	AzureProviderSpecWithEmptyFieldImageURN = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithNegativeOSDiskSize with Negative Disk size
	AzureProviderSpecWithNegativeOSDiskSize = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   -50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithoutOSDiskCreateOption without OS Disk Creation Option
	AzureProviderSpecWithoutOSDiskCreateOption = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithoutAdminUserName without Admin Username
	AzureProviderSpecWithoutAdminUserName = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
			},
			OsProfile: api.AzureOSProfile{
				AdminUsername: "",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithoutZMA with Zone, MachineSet and Availability
	AzureProviderSpecWithoutZMA = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: nil,
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithZMA with Zone, MachineSet and AvailabilitySet
	AzureProviderSpecWithZMA = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
			},
			MachineSet: &api.AzureMachineSetConfig{
				ID:   "example-id",
				Kind: "vmo",
			},
			AvailabilitySet: &api.AzureSubResource{
				ID: "example-id",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithMAOnly with MachineSet and AvailabilitySet only and no Zone
	AzureProviderSpecWithMAOnly = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
			},
			MachineSet: &api.AzureMachineSetConfig{
				ID:   "example-id",
				Kind: "vmo",
			},
			AvailabilitySet: &api.AzureSubResource{
				ID: "example-id",
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
			Zone: nil,
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithInvalidMachineSet with Invlaid MachineSet Kind
	AzureProviderSpecWithInvalidMachineSet = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
			},
			MachineSet: &api.AzureMachineSetConfig{
				ID:   "example-id",
				Kind: "machineSet",
			},
			AvailabilitySet: &api.AzureSubResource{
				ID: "example-id",
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
			Zone: nil,
		},
		ResourceGroup: "",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithEmptyClusterNameTag with Empty Cluster Name Tag
	AzureProviderSpecWithEmptyClusterNameTag = api.AzureProviderSpec{
		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "node",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithEmptyNodeRoleTag with Empty Node Role Tag
	AzureProviderSpecWithEmptyNodeRoleTag = api.AzureProviderSpec{

		Location: "westeurope",
		Properties: api.AzureVirtualMachineProperties{
			HardwareProfile: api.AzureHardwareProfile{
				VMSize: "Standard_DS2_v2",
			},
			StorageProfile: api.AzureStorageProfile{
				ImageReference: api.AzureImageReference{
					URN: getUrn("sap:gardenlinux:greatest:27.1.0"),
				},
				OsDisk: api.AzureOSDisk{
					Caching: "None",
					ManagedDisk: api.AzureManagedDiskParameters{
						StorageAccountType: "Standard_LRS",
					},
					DiskSizeGB:   50,
					CreateOption: "FromImage",
				},
				DataDisks: []api.AzureDataDisk{},
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--i538135--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--i538135--seed-az",
			SubnetName: "shoot--i538135--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--i538135--seed-az",
			"kubernetes.io-cluster-shoot--i538135--seed-az": "1",
			"kubernetes.io-role-mcm":                        "1",
			"node.kubernetes.io_role":                       "",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}
)
