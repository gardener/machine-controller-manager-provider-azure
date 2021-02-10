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

	// // This is the AzureProviderSpec without location value
	// AzureProviderSpecWithoutLocation = []byte("{\"location\":\"\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the AzureProviderSpec without resource group value
	// AzureProviderSpecWithoutResourceGroup = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the AzureProviderSpec without resource group value
	// AzureProviderSpecWithoutVnetName = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of ProviderSpec key of Kind Machine Class for Azure
	// AzureProviderSpecWithoutSubnetName = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of the Azure ProviderSpec without VMSize
	// AzureProviderSpecWithoutVMSize = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of the Azure ProviderSpec without ImageURN
	// AzureProviderSpecWithoutImageURN = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of the Azure ProviderSpec with Improper ImageURN
	// AzureProviderSpecWithImproperImageURN = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sapgardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of the Azure ProviderSpec with EmptyField Image URN
	// AzureProviderSpecWithEmptyFieldImageURN = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\":gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of the Azure ProviderSpec with Negative Disk size
	// AzureProviderSpecWithNegativeOSDiskSize = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":-50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of the Azure ProviderSpec without OS Disk Creation Option
	// AzureProviderSpecWithoutOSDiskCreateOption = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of the Azure ProviderSpec without Admin Username
	// AzureProviderSpecWithoutAdminUserName = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the value of the Azure ProviderSpec with Zone, MachineSet and Availability
	// AzureProviderSpecWithoutZMA = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}}},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the AzureProviderSpec with Zone, MachineSet and AvailabilitySet
	// AzureProviderSpecWithZMA = []byte("{\"location\":\"westeurope\",\"properties\":{\"availabilitySet\":{\"id\":\"\"},\"machineSet\":{\"ID\":\"ID\",\"kind\":\"machinekind\"},\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the AzureProviderSpec with MachineSet and AvailabilitySet only and no Zone
	// AzureProviderSpecWithMAOnly = []byte("{\"location\":\"westeurope\",\"properties\":{\"availabilitySet\":{\"id\":\"some-id\"},\"machineSet\":{\"ID\":\"ID\",\"kind\":\"vmo\"},\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}}},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the AzureProviderSpec with Invlaid MachineSet Kind
	// AzureProviderSpecWithInvalidMachineSet = []byte("{\"location\":\"westeurope\",\"properties\":{\"machineSet\":{\"ID\":\"ID\",\"kind\":\"machinekind\"},\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}}},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the Azure ProviderSpec with Empty Cluster Name Tag
	// AzureProviderSpecWithEmptyClusterNameTag = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-role-mcm\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")

	// // This is the Azure ProviderSpec with Empty Node Role Tag
	// AzureProviderSpecWithEmptyNodeRoleTag = []byte("{\"location\":\"westeurope\",\"properties\":{\"hardwareProfile\":{\"vmSize\":\"Standard_DS2_v2\"},\"osProfile\":{\"adminUsername\":\"core\",\"linuxConfiguration\":{\"disablePasswordAuthentication\":true,\"ssh\":{\"publicKeys\":{\"keyData\":\"dummy keyData\",\"path\":\"/home/core/.ssh/authorized_keys\"}}}},\"storageProfile\":{\"imageReference\":{\"urn\":\"sap:gardenlinux:greatest:27.1.0\"},\"osDisk\":{\"caching\":\"None\",\"createOption\":\"FromImage\",\"diskSizeGB\":50,\"managedDisk\":{\"storageAccountType\":\"Standard_LRS\"}}},\"zone\":2},\"resourceGroup\":\"shoot--i538135--seed-az\",\"subnetInfo\":{\"subnetName\":\"shoot--i538135--seed-az-nodes\",\"vnetName\":\"shoot--i538135--seed-az\"},\"tags\":{\"Name\":\"shoot--i538135--seed-az\",\"kubernetes.io-cluster-shoot--i538135--seed-az\":\"1\",\"node.kubernetes.io_role\":\"node\",\"worker.garden.sapcloud.io_group\":\"worker-m0exd\",\"worker.gardener.cloud_pool\":\"worker-m0exd\",\"worker.gardener.cloud_system-components\":\"true\"}}")
)
