/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package validation - validation is used to validate cloud specific ProviderSpec
package validation

import (
	"fmt"
	"regexp"
	"strings"

	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"

	corev1 "k8s.io/api/core/v1"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const nameFmt string = `[-a-z0-9]+`

var nameRegexp = regexp.MustCompile("^" + nameFmt + "$")

// ValidateAzureSpecNSecret validates Azure provider spec
func ValidateAzureSpecNSecret(spec *api.AzureProviderSpec, secrets *corev1.Secret) []error {
	var allErrs []error

	if "" == spec.Location {
		allErrs = append(allErrs, fmt.Errorf("Region is required field"))
	}
	if "" == spec.ResourceGroup {
		allErrs = append(allErrs, fmt.Errorf("ResourceGroup is required field"))
	}

	allErrs = append(allErrs, validateSpecSubnetInfo(spec.SubnetInfo)...)
	allErrs = append(allErrs, validateSpecProperties(spec.Properties)...)
	allErrs = append(allErrs, validateSecrets(secrets)...)
	allErrs = append(allErrs, validateSpecTags(spec.Tags)...)

	return allErrs
}

func validateSpecSubnetInfo(subnetInfo api.AzureSubnetInfo) []error {
	var allErrs []error

	if "" == subnetInfo.VnetName {
		allErrs = append(allErrs, fmt.Errorf("VnetName is a required subnet info"))
	}
	if "" == subnetInfo.SubnetName {
		allErrs = append(allErrs, fmt.Errorf("SubnetName is a required subnet info"))
	}

	return allErrs
}

func validateSpecProperties(properties api.AzureVirtualMachineProperties) []error {
	var allErrs []error

	var fldPath *field.Path

	fldPath = field.NewPath("properties")

	if properties.HardwareProfile.VMSize == "" {
		allErrs = append(allErrs, fmt.Errorf("VMSize is required field"))
	}

	allErrs = append(allErrs, ValidateImageReference(properties.StorageProfile.ImageReference, fldPath.Child("storageProfile.imageReference"))...)

	if properties.StorageProfile.OsDisk.DiskSizeGB <= 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("storageProfile.osDisk.diskSizeGB"), "OSDisk size must be positive"))
	}
	if properties.StorageProfile.OsDisk.CreateOption == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("storageProfile.osDisk.createOption"), "OSDisk create option is required"))
	}

	if properties.StorageProfile.DataDisks != nil {

		if len(properties.StorageProfile.DataDisks) > 64 {
			allErrs = append(allErrs, field.TooMany(fldPath.Child("storageProfile.dataDisks"), len(properties.StorageProfile.DataDisks), 64))
		}

		luns := map[int32]int{}
		for i, dataDisk := range properties.StorageProfile.DataDisks {
			idxPath := fldPath.Child("storageProfile.dataDisks").Index(i)

			lun := dataDisk.Lun

			if lun == nil {
				allErrs = append(allErrs, field.Required(idxPath.Child("lun"), "DataDisk Lun is required"))
			} else {
				if *lun < 0 || *lun > 63 {
					allErrs = append(allErrs, field.Invalid(idxPath.Child("lun"), *lun, utilvalidation.InclusiveRangeError(0, 63)))
				}
				if _, keyExist := luns[*lun]; keyExist {
					luns[*lun]++
				} else {
					luns[*lun] = 1
				}
			}

			if dataDisk.DiskSizeGB <= 0 {
				allErrs = append(allErrs, field.Required(idxPath.Child("diskSizeGB"), "DataDisk size must be positive"))
			}
			if dataDisk.StorageAccountType == "" {
				allErrs = append(allErrs, field.Required(idxPath.Child("storageAccountType"), "DataDisk storage account type is required"))
			}
		}

		for lun, number := range luns {
			if number > 1 {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("storageProfile.dataDisks"), lun, fmt.Sprintf("Data Disk Lun '%d' duplicated %d times, Lun must be unique", lun, number)))
			}
		}
	}
	if properties.OsProfile.AdminUsername == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("osProfile.adminUsername"), "AdminUsername is required"))
	}

	if properties.Zone == nil && properties.MachineSet == nil && properties.AvailabilitySet == nil {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("zone|.machineSet|.availabilitySet"), "Machine need to be assigned to a zone, a MachineSet or an AvailabilitySet"))
	}

	if properties.Zone != nil && (properties.MachineSet != nil || properties.AvailabilitySet != nil) {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("zone|.machineSet|.availabilitySet"), "Machine cannot be assigned to a zone, a MachineSet and an AvailabilitySet in parallel"))
	}

	if properties.Zone == nil {
		if properties.MachineSet != nil && properties.AvailabilitySet != nil {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("machineSet|.availabilitySet"), "Machine cannot be assigned a MachineSet and an AvailabilitySet in parallel"))
		}
		if properties.MachineSet != nil && !(properties.MachineSet.Kind == api.MachineSetKindVMO || properties.MachineSet.Kind == api.MachineSetKindAvailabilitySet) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("machineSet"), properties.MachineSet.Kind, fmt.Sprintf("Invalid MachineSet kind. Use either '%s' or '%s'", api.MachineSetKindVMO, api.MachineSetKindAvailabilitySet)))
		}
	}

	return allErrs
}

func validateSpecTags(tags map[string]string) []error {

	var fldPath *field.Path
	var allErrs []error

	fldPath = field.NewPath("providerSpec")
	clusterName := ""
	nodeRole := ""

	for key := range tags {
		if strings.Contains(key, "kubernetes.io-cluster-") {
			clusterName = key
		} else if strings.Contains(key, "kubernetes.io-role-") {
			nodeRole = key
		}
	}

	if clusterName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("kubernetes.io-cluster-"), "Tag required of the form kubernetes.io-cluster-****"))
	}
	if nodeRole == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("kubernetes.io-role-"), "Tag required of the form kubernetes.io-role-****"))
	}

	return allErrs
}

func validateSecrets(secret *corev1.Secret) []error {
	var allErrs []error

	if "" == string(secret.Data[api.AzureClientID]) && "" == string(secret.Data[api.AzureAlternativeClientID]) {
		allErrs = append(allErrs, fmt.Errorf("secret %s or %s is required field", api.AzureClientID, api.AzureAlternativeClientID))
	}
	if "" == string(secret.Data[api.AzureClientSecret]) && "" == string(secret.Data[api.AzureAlternativeClientSecret]) {
		allErrs = append(allErrs, fmt.Errorf("secret %s or %s is required field", api.AzureClientSecret, api.AzureAlternativeClientSecret))
	}
	if "" == string(secret.Data[api.AzureTenantID]) && "" == string(secret.Data[api.AzureAlternativeTenantID]) {
		allErrs = append(allErrs, fmt.Errorf("secret %s or %s is required field", api.AzureTenantID, api.AzureAlternativeTenantID))
	}
	if "" == string(secret.Data[api.AzureSubscriptionID]) && "" == string(secret.Data[api.AzureAlternativeSubscriptionID]) {
		allErrs = append(allErrs, fmt.Errorf("secret %s or %s is required field", api.AzureSubscriptionID, api.AzureAlternativeSubscriptionID))
	}
	if "" == string(secret.Data["userData"]) {
		allErrs = append(allErrs, fmt.Errorf("secret UserData is required field"))
	}
	return allErrs
}

// ValidateImageReference is validating the image reference config.
// TODO Do not export this function anymore once proper unit test coverage is established
// for ValidateAzureSpecNSecret().
func ValidateImageReference(imageRef api.AzureImageReference, fldPath *field.Path) []error {
	var allErrs []error

	if isEmptyStringPtr(imageRef.URN) && isEmptyStringPtr(imageRef.CommunityGalleryImageID) && isEmptyString(imageRef.ID) && isEmptyStringPtr(imageRef.SharedGalleryImageID) {
		return append(allErrs, field.Required(fldPath, "must specify either a image id, community gallery image id, shared gallery image id or an urn"))
	}

	if !isEmptyStringPtr(imageRef.URN) {
		if !isEmptyStringPtr(imageRef.CommunityGalleryImageID) || !isEmptyString(imageRef.ID) || !isEmptyStringPtr(imageRef.SharedGalleryImageID) {
			return append(allErrs, field.Required(fldPath.Child("urn"), "cannot specify community gallery image id, shared gallery image id or image id when urn is specified"))
		}

		urnParts := strings.Split(*imageRef.URN, ":")
		if len(urnParts) != 4 {
			return append(allErrs, field.Required(fldPath.Child("urn"), "invalid urn format"))
		}

		for _, s := range urnParts {
			if isEmptyString(s) {
				allErrs = append(allErrs, field.Required(fldPath.Child("urn"), "invalid urn format, empty field"))
			}
		}

		return allErrs
	}

	if !isEmptyStringPtr(imageRef.CommunityGalleryImageID) {
		if !isEmptyString(imageRef.ID) || !isEmptyStringPtr(imageRef.SharedGalleryImageID) {
			return append(allErrs, field.Required(fldPath.Child("communityGalleryImageID"), "cannot specify shared gallery image id or image id when community gallery image id is specified"))
		}

		return allErrs
	}

	if !isEmptyStringPtr(imageRef.SharedGalleryImageID) {
		if !isEmptyString(imageRef.ID) {
			return append(allErrs, field.Required(fldPath.Child("sharedGalleryImageID"), "cannot specify a shared image gallery id and image id in parallel"))
		}
		return allErrs
	}

	if isEmptyString(imageRef.ID) {
		return append(allErrs, field.Required(fldPath.Child("id"), "invalid image id format, empty field"))
	}

	return allErrs
}

func isEmptyString(s string) bool {
	if s == "" {
		return true
	}
	return false
}

func isEmptyStringPtr(s *string) bool {
	if s == nil {
		return true
	}
	return isEmptyString(*s)
}
