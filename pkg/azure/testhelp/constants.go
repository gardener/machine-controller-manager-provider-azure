// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testhelp

const (
	SubscriptionID     = "test-subscription-id"
	TenantID           = "test-tenant"
	ClientID           = "test-client-id"
	ClientSecret       = "test-client-secret"
	StorageAccountType = "StandardSSD_LRS"
	VMSize             = "Standard_DS2_v2"
	Location           = "test-west-euro"
	DefaultImageRefURN = "sap:gardenlinux:greatest:184.0.0"
	TestAdminUserName  = "core"
)

// Constants for method names for different fake servers. These will be used by consumers to set API behavior on specific methods.
const (
	AccessMethodGet                 = "Get"
	AccessMethodCreate              = "Create"
	AccessMethodBeginDelete         = "BeginDelete"
	AccessMethodBeginUpdate         = "BeginUpdate"
	AccessMethodCheckExistence      = "CheckExistence"
	AccessMethodBeginCreateOrUpdate = "BeingCreateOrUpdate"
)

// Constants are used to parse the request URI Path and represent the resource types

const (
	ResourceTypeSubnet           = "subnets"
	ResourceTypeVirtualMachine   = "virtualMachines"
	ResourceTypeNetworkInterface = "networkInterfaces"
)
