package testhelp

// Error codes
const (
	ErrorCodeResourceNotFound      = "ResourceNotFound"
	ErrorCodeResourceGroupNotFound = "ResourceGroupNotFound"
	ErrorCodePatchResourceNotFound = "PatchResourceNotFound"
	ErrorOperationNotAllowed       = "OperationNotAllowed"
	ErrorBadRequest                = "BadRequest"
)

const (
	SubscriptionID     = "test-subscription-id"
	TenantID           = "test-tenant"
	ClientID           = "test-client-id"
	ClientSecret       = "test-client-secret"
	StorageAccountType = "StandardSSD_LRS"
	VMSize             = "Standard_DS2_v2"
	Location           = "test-west-euro"
	ImageRefURN        = "sap:gardenlinux:greatest:184.0.0"
	TestAdminUserName  = "core"
)

const (
	AccessMethodGet            = "Get"
	AccessMethodBeginDelete    = "BeginDelete"
	AccessMethodBeginUpdate    = "BeginUpdate"
	AccessMethodCheckExistence = "CheckExistence"
)