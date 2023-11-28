package utils

// ResourceType is an enum type representing type different resource types supported by Azure.
type ResourceType string

const (
	// VirtualMachinesResourceType is a type used by Azure to represent virtual machine resources.
	VirtualMachinesResourceType ResourceType = "microsoft.compute/virtualmachines"
	// NetworkInterfacesResourceType is a type used by Azure to represent network interfaces resources.
	NetworkInterfacesResourceType ResourceType = "microsoft.network/networkinterfaces" //as defined in azure
	// DiskResourceType is a type used by Azure to represent disks (both OS and Data disks)
	DiskResourceType ResourceType = "microsoft.compute/disks"
	// VMImageResourceType is a type used by Azure to represent VM Image resources.
	// This is not defined in azure, however we have created this to allow defining API behavior for VM Images.
	VMImageResourceType ResourceType = "microsoft.compute/vmimage"
	// MarketPlaceOrderingOfferType is a type used by Azure to represent marketplace ordering offer types.
	MarketPlaceOrderingOfferType ResourceType = "microsoft.marketplaceordering/offertypes"
	// SubnetResourceType is a type used by Azure to represent subnet resources.
	SubnetResourceType ResourceType = "microsoft.network/virtualnetworks/subnets"
)
