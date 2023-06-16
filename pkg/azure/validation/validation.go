/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package validation - validation is used to validate cloud specific ProviderSpec
package validation

import (
	"fmt"
	"strings"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const providerAzure = "Azure"

// ValidateMachineClassProvider checks if the Provider in MachineClass is Azure.
// If it is not then it will return an error indicating that this provider implementation cannot fulfill the request.
func ValidateMachineClassProvider(mcc *v1alpha1.MachineClass) error {
	if mcc.Provider != providerAzure {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("Request for provider %s cannot be fulfilled. Only %s provider is supported.", mcc.Provider, providerAzure))
	}
	return nil
}

// ValidateProviderSpec validates the api.AzureProviderSpec.
func ValidateProviderSpec(spec *api.AzureProviderSpec) field.ErrorList {
	allErrs := field.ErrorList{}
	specPath := field.NewPath("providerSpec")

	if len(strings.TrimSpace(spec.Location)) == 0 {
		allErrs = append(allErrs, field.Required(specPath.Child("location"), "must provide a location"))
	}
	if len(strings.TrimSpace(spec.ResourceGroup)) == 0 {
		allErrs = append(allErrs, field.Required(specPath.Child("resourceGroup"), "must provide a resourceGroup"))
	}

	allErrs = append(allErrs, validateSubnetInfo(spec.SubnetInfo, specPath.Child("subnetInfo"))...)
	allErrs = append(allErrs, validateProperties(spec.Properties, specPath.Child("properties"))...)
	allErrs = append(allErrs, validateTags(spec.Tags, specPath.Child("tags"))...)

	return allErrs
}

// ValidateProviderSecret validates the secret containing the config to create Azure API clients.
func ValidateProviderSecret(secret *corev1.Secret) field.ErrorList {
	allErrs := field.ErrorList{}
	secretDataPath := field.NewPath("data")
	if utils.IsEmptyString(string(secret.Data[api.ClientID])) && utils.IsEmptyString(string(secret.Data[api.AzureClientID])) && utils.IsEmptyString(string(secret.Data[api.AzureAlternativeClientID])) {
		allErrs = append(allErrs, field.Required(secretDataPath.Child("clientID"), "must provide clientID"))
	}

	if utils.IsEmptyString(string(secret.Data[api.ClientSecret])) && utils.IsEmptyString(string(secret.Data[api.AzureClientSecret])) && utils.IsEmptyString(string(secret.Data[api.AzureAlternativeClientSecret])) {
		allErrs = append(allErrs, field.Required(secretDataPath.Child("clientSecret"), "must provide clientSecret"))
	}

	if utils.IsEmptyString(string(secret.Data[api.SubscriptionID])) && utils.IsEmptyString(string(secret.Data[api.AzureSubscriptionID])) && utils.IsEmptyString(string(secret.Data[api.AzureAlternativeSubscriptionID])) {
		allErrs = append(allErrs, field.Required(secretDataPath.Child("subscriptionID"), "must provide subscriptionID"))
	}

	if utils.IsEmptyString(string(secret.Data[api.TenantID])) && utils.IsEmptyString(string(secret.Data[api.AzureTenantID])) && utils.IsEmptyString(string(secret.Data[api.AzureAlternativeTenantID])) {
		allErrs = append(allErrs, field.Required(secretDataPath.Child("tenantID"), "must provide tenantID"))
	}

	return allErrs
}

// ValidateMachineSetConfig validates the now deprecated api.AzureMachineSetConfig. This method should be removed once all
// consumers have migrated away from using this field and moved completely to either api.AzureVirtualMachineProperties.AvailabilitySet
// or AzureVirtualMachineProperties.VirtualMachineScaleSet
func ValidateMachineSetConfig(machineSetConfig *api.AzureMachineSetConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	fldPath := field.NewPath("providerSpec.properties.machineSet")
	allowedKinds := sets.New(api.MachineSetKindAvailabilitySet, api.MachineSetKindVMO)
	if machineSetConfig != nil && !allowedKinds.Has(machineSetConfig.Kind) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("kind"), machineSetConfig.Kind, fmt.Sprintf("must provide one of %v", allowedKinds)))
	}
	return allErrs
}

func validateSubnetInfo(subnetInfo api.AzureSubnetInfo, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if utils.IsEmptyString(subnetInfo.VnetName) {
		allErrs = append(allErrs, field.Required(fldPath.Child("vnetName"), "must provide vnetName"))
	}
	if utils.IsEmptyString(subnetInfo.SubnetName) {
		allErrs = append(allErrs, field.Required(fldPath.Child("subnetName"), "must provide subnetName"))
	}

	return allErrs
}

func validateProperties(properties api.AzureVirtualMachineProperties, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// validate HardwareProfile
	if utils.IsEmptyString(properties.HardwareProfile.VMSize) {
		allErrs = append(allErrs, field.Required(fldPath.Child("vmSize"), "must provide vmSize"))
	}
	// validate StorageProfile
	allErrs = append(allErrs, validateStorageProfile(properties.StorageProfile, fldPath.Child("storageProfile"))...)
	// validate OSProfile
	if utils.IsEmptyString(properties.OsProfile.AdminUsername) {
		allErrs = append(allErrs, field.Required(fldPath.Child("osProfile.adminUsername"), "adminUsername must be provided"))
	}
	allErrs = append(allErrs, validateAvailabilityAndScalingConfig(properties, fldPath)...)

	return allErrs
}

func validateStorageProfile(storageProfile api.AzureStorageProfile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateStorageImageRef(storageProfile.ImageReference, fldPath.Child("imageReference"))...)
	allErrs = append(allErrs, validateOSDisk(storageProfile.OsDisk, fldPath.Child("osDisk"))...)
	allErrs = append(allErrs, validateDataDisks(storageProfile.DataDisks, fldPath.Child("dataDisks"))...)
	return allErrs
}

func validateStorageImageRef(imageRef api.AzureImageReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	urnIsSet := !utils.IsNilAndEmptyStringPtr(imageRef.URN)
	communityGalleryImageIDIsSet := !utils.IsNilAndEmptyStringPtr(imageRef.CommunityGalleryImageID)
	idIsSet := !utils.IsEmptyString(imageRef.ID)
	sharedGalleryImageIDIsSet := !utils.IsNilAndEmptyStringPtr(imageRef.SharedGalleryImageID)

	atMostOnceIdentifierSet := atMostOneShouldBeTrue(urnIsSet, communityGalleryImageIDIsSet, idIsSet, sharedGalleryImageIDIsSet)
	if !atMostOnceIdentifierSet {
		return append(allErrs, field.Forbidden(fldPath.Child("id|.urn|.communityGalleryImageID|.sharedGalleryImageID"), "must specify only one of image id, community gallery image id, shared gallery image id or an urn"))
	}

	if urnIsSet {
		allErrs = append(allErrs, validateURN(*imageRef.URN, fldPath.Child("urn"))...)
		return allErrs
	}

	return allErrs
}

func validateOSDisk(osDisk api.AzureOSDisk, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if utils.IsEmptyString(osDisk.CreateOption) {
		allErrs = append(allErrs, field.Required(fldPath.Child("createOption"), "must provide createOption"))
	}
	if osDisk.DiskSizeGB <= 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("diskSizeGB"), osDisk.DiskSizeGB, "OSDisk size must be positive and greater than 0"))
	}
	return allErrs
}

func validateDataDisks(disks []api.AzureDataDisk, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if disks == nil {
		return allErrs
	}

	luns := make(map[int32]int, len(disks))
	for _, disk := range disks {
		if disk.Lun == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("lun"), "must provide lun"))
		} else {
			if *disk.Lun < 0 {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("lun"), *disk.Lun, "lun must be a positive number"))
			} else {
				luns[*disk.Lun]++
			}
		}
		if disk.DiskSizeGB <= 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("diskSizeGB"), disk.DiskSizeGB, "DataDisk size must be positive and greater than 0"))
		}
		if utils.IsEmptyString(disk.StorageAccountType) {
			allErrs = append(allErrs, field.Required(fldPath.Child("storageAccountType"), "must provide storageAccountType"))
		}
	}

	for lun, numOccurrence := range luns {
		if numOccurrence > 1 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("lun"), lun, fmt.Sprintf("DataDisk Lun '%d' duplicated %d times, Lun must be unique", lun, numOccurrence)))
		}
	}

	return allErrs
}

func validateAvailabilityAndScalingConfig(properties api.AzureVirtualMachineProperties, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	isZoneConfigured := properties.Zone != nil
	isAvailabilitySetConfigured := properties.AvailabilitySet != nil && !utils.IsEmptyString(properties.AvailabilitySet.ID)
	isVirtualMachineScaleSetConfigured := properties.VirtualMachineScaleSet != nil && !utils.IsEmptyString(properties.VirtualMachineScaleSet.ID)

	// check if both zone is configured and one or both of [availabilitySet, virtualMachineScaleSet] is configured
	if !atMostOneShouldBeTrue(isZoneConfigured, isAvailabilitySetConfigured || isVirtualMachineScaleSetConfigured) {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("zone|.availabilitySet|.virtualMachineScaleSet"), "Either a Zone can be specified or one of (AvailabilitySet or VirtualMachineScaleSet) can be set."))
	}

	// if zone is not configured then check that only one of [availabilitySet, virtualMachineScaleSet] is configured
	if properties.Zone == nil && !atMostOneShouldBeTrue(isAvailabilitySetConfigured, isVirtualMachineScaleSetConfigured) {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("availabilitySet|.virtualMachineScaleSet"), "Must only configure an AvailabilitySet or VirtualMachineScaleSet but not both."))
	}

	return allErrs
}

func validateTags(tags map[string]string, fldPath *field.Path) field.ErrorList {
	const (
		clusterKeyPrefix  = "kubernetes.io-cluster-"
		nodeRoleKeyPrefix = "kubernetes.io-role-"
	)
	allErrs := field.ErrorList{}
	if tags == nil {
		return append(allErrs, field.Required(fldPath.Child(clusterKeyPrefix, nodeRoleKeyPrefix), fmt.Sprintf("Tags starting with '%s' and '%s' must be set", clusterKeyPrefix, nodeRoleKeyPrefix)))
	}

	var clusterKeySet, nodeRoleKeySet bool
	for key := range tags {
		if strings.HasPrefix(key, clusterKeyPrefix) {
			clusterKeySet = true
		} else if strings.HasPrefix(key, nodeRoleKeyPrefix) {
			nodeRoleKeySet = true
		}
	}
	if !clusterKeySet {
		allErrs = append(allErrs, field.Required(fldPath.Child(clusterKeyPrefix), fmt.Sprintf("Tag starting with %s must be set", clusterKeyPrefix)))
	}
	if !nodeRoleKeySet {
		allErrs = append(allErrs, field.Required(fldPath.Child(nodeRoleKeyPrefix), fmt.Sprintf("Tag starting with %s must be set", nodeRoleKeyPrefix)))
	}

	return allErrs
}

// validateURN validates if the URN format is as required by azure.
// URN has the following format: <Publisher>:<Offer>:<SKU>:<Version>
// The details of each part is as follows:
// Publisher: The organization that created the image. Examples: Canonical, MicrosoftWindowsServer
// Offer: The name of a group of related images created by a publisher. Examples: UbuntuServer, WindowsServer
// SKU: An instance of an offer, such as a major release of a distribution. Examples: 18.04-LTS, 2019-Datacenter
// Version: The version number of an image SKU.
func validateURN(urn string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	urnParts := strings.Split(urn, ":")
	if len(urnParts) != 4 {
		return append(allErrs, field.Invalid(fldPath, urn, "invalid urn format, urn should be of the format publisher:offer:sku:version"))
	}
	urnPartLabels := []string{"publisher", "offer", "sku", "version"}
	for index, urnPart := range urnParts {
		if utils.IsEmptyString(urnPart) {
			allErrs = append(allErrs, field.Required(fldPath, fmt.Sprintf("urn must have %s", urnPartLabels[index])))
		}
	}

	return allErrs
}

func atMostOneShouldBeTrue(conditions ...bool) bool {
	prevCondition := false
	for _, c := range conditions {
		if c && prevCondition {
			return false
		}
		prevCondition = c || prevCondition
	}
	return prevCondition
}
