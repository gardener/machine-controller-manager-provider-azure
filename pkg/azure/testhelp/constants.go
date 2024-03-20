// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testhelp

const (
	// SubscriptionID is the test azure subscription ID.
	SubscriptionID = "test-subscription-id"
	// TenantID is the test azure tenant ID.
	TenantID = "test-tenant"
	// ClientID is the test azure client ID.
	ClientID = "test-client-id"
	// ClientSecret is the test azure client secret.
	ClientSecret = "test-client-secret"
	// StorageAccountType is the test azure storage account type.
	StorageAccountType = "StandardSSD_LRS"
	// VMSize is the test azure VM type.
	VMSize = "Standard_DS2_v2"
	// Location is the test azure location.
	Location = "test-west-euro"
	// DefaultImageRefURN is the test azure image URN.
	DefaultImageRefURN = "sap:gardenlinux:greatest:184.0.0"
	// UserData is the dummy user data that is set as part of the secret
	UserData = "dummy-user-data"
)

// Constants for method names for different fake servers. These will be used by consumers to set API behavior on specific methods.
const (
	// AccessMethodGet is the constant representing Get Azure API method name in the fake server.
	AccessMethodGet = "Get"
	// AccessMethodCreate is the constant representing Create Azure API method name in the fake server.
	AccessMethodCreate = "Create"
	// AccessMethodBeginDelete is the constant representing BeginDelete Azure API method name in the fake server.
	AccessMethodBeginDelete = "BeginDelete"
	// AccessMethodBeginUpdate is the constant representing BeginUpdate Azure API method name in the fake server.
	AccessMethodBeginUpdate = "BeginUpdate"
	// AccessMethodCheckExistence is the constant representing CheckExistence Azure API method name in the fake server.
	AccessMethodCheckExistence = "CheckExistence"
	// AccessMethodBeginCreateOrUpdate is the constant representing BeginCreateOrUpdate Azure API method name in the fake server.
	AccessMethodBeginCreateOrUpdate = "BeginCreateOrUpdate"
	// AccessMethodResources is the constant representing Resources Azure API method name in the fake server.
	AccessMethodResources = "Resources"
)
