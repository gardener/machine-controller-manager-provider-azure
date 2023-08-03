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
