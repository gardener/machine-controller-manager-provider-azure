// Package mock has the mock framework of Azure SDK for Go for unit testing
package mock

/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

import api "github.com/gardener/machine-controller-manager-provider-azure/pkg/apis/v1"

func getUrn(urn string) *string {
	return &urn
}

func getZone(zone int) *int {
	return &zone
}

func getInt32Pointer(i int32) *int32 {
	return &i
}

var tags = map[string]string{
	"Name":                                    "shoot--project",
	"kubernetes.io-cluster-shoot--project":    "1",
	"kubernetes.io-role-mcm":                  "1",
	"node.kubernetes.io_role":                 "node",
	"worker.garden.sapcloud.io_group":         "worker-m0exd",
	"worker.gardener.cloud_pool":              "worker-m0exd",
	"worker.gardener.cloud_system-components": "true",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
	}

	// AzureProviderSpecWithNegativeDataDiskSize with Negative Disk size
	AzureProviderSpecWithNegativeDataDiskSize = api.AzureProviderSpec{
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
				DataDisks: []api.AzureDataDisk{
					{
						Lun:                getInt32Pointer(1),
						DiskSizeGB:         -50,
						StorageAccountType: "Standard_LRS",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
	}

	// AzureProviderSpecWithoutLUN with Negative Disk size
	AzureProviderSpecWithoutLUN = api.AzureProviderSpec{
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
				DataDisks: []api.AzureDataDisk{
					{
						Lun:                nil,
						DiskSizeGB:         50,
						StorageAccountType: "Standard_LRS",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
	}

	// AzureProviderSpecWithImproperLUN with Negative Disk size
	AzureProviderSpecWithImproperLUN = api.AzureProviderSpec{
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
				DataDisks: []api.AzureDataDisk{
					{
						Lun:                getInt32Pointer(-1),
						DiskSizeGB:         50,
						StorageAccountType: "Standard_LRS",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
	}

	// AzureProviderSpecWithoutDiskStorageAccountType with Negative Disk size
	AzureProviderSpecWithoutDiskStorageAccountType = api.AzureProviderSpec{
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
				DataDisks: []api.AzureDataDisk{
					{
						Lun:                getInt32Pointer(1),
						DiskSizeGB:         50,
						StorageAccountType: "",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
	}

	// AzureProviderSpecWithDuplicatedLUN with Negative Disk size
	AzureProviderSpecWithDuplicatedLUN = api.AzureProviderSpec{
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
				DataDisks: []api.AzureDataDisk{
					{
						Lun:                getInt32Pointer(1),
						DiskSizeGB:         50,
						StorageAccountType: "Standard_LRS",
					},
					{
						Lun:                getInt32Pointer(1),
						DiskSizeGB:         50,
						StorageAccountType: "Standard_LRS",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
				DataDisks: []api.AzureDataDisk{
					{
						Name:               "data-disk",
						Lun:                getInt32Pointer(1),
						StorageAccountType: "Standard_LRS",
						DiskSizeGB:         50,
					},
				},
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
			AvailabilitySet: nil,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name":                                    "",
			"kubernetes.io-role-mcm":                  "1",
			"node.kubernetes.io_role":                 "node",
			"worker.garden.sapcloud.io_group":         "worker-m0exd",
			"worker.gardener.cloud_pool":              "worker-m0exd",
			"worker.gardener.cloud_system-components": "true",
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
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: map[string]string{
			"Name": "shoot--project--seed-az",
			"kubernetes.io-cluster-shoot--project--seed-az": "1",
			"node.kubernetes.io_role":                       "",
			"worker.garden.sapcloud.io_group":               "worker-m0exd",
			"worker.gardener.cloud_pool":                    "worker-m0exd",
			"worker.gardener.cloud_system-components":       "true",
		},
	}

	// AzureProviderSpecWithDataDisks with providerSpec that has data disk attached
	AzureProviderSpecWithDataDisks = api.AzureProviderSpec{
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
	}

	// AzureProviderSpecWithDataDisksWithName with providerSpec that has data disk attached
	AzureProviderSpecWithDataDisksWithName = api.AzureProviderSpec{
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
				DataDisks: []api.AzureDataDisk{
					{
						StorageAccountType: "Standard_LRS",
						Lun:                getInt32Pointer(1),
						DiskSizeGB:         50,
						Name:               "data-disk",
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
			Zone: getZone(2),
		},
		ResourceGroup: "shoot--project--seed-az",
		SubnetInfo: api.AzureSubnetInfo{
			VnetName:   "shoot--project--seed-az",
			SubnetName: "shoot--project--seed-az-nodes",
		},
		Tags: tags,
	}
)
