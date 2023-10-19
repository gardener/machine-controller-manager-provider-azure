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

package validation

import (
	"encoding/base64"
	"testing"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

func TestValidateProviderSecret(t *testing.T) {
	const (
		/*
			Trivia: On posix shell you can use the following to generate uuid and random alphanumeric strings:
				* To generate uuid use: `uuidgen | awk '{print tolower($0)}'`
			    * To generate random string using allowed characters and specified length use: `cat /dev/urandom | LC_ALL=C tr -dc 'a-zA-Z0-9~' | fold -w 50 | head -n 1`
		*/
		testClientID       = "c9f8e78f-eba7-4d2d-97fe-ea4679dbbe63"
		testClientSecret   = "to6D2mXsZ~lNJsUi0H5lZsRgrh7FlWMTXdTfeKaMO8fCbKmUYE"
		testSubscriptionID = "8edcc1ad-04bc-419c-ad63-1a989956d466"
		testTenantID       = "010bd0ff-5eae-446e-aea9-c1eac72e9c77"
		testUserData       = "May the force be with you"
	)

	table := []struct {
		description    string
		clientID       string
		clientSecret   string
		subscriptionID string
		tenantID       string
		testUserData   string
		expectedErrors int
		matcher        gomegatypes.GomegaMatcher
	}{
		{
			"should forbid empty clientID",
			"", testClientSecret, testSubscriptionID, testTenantID, testUserData, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.clientID")}))),
		},
		// just testing one field with spaces. handling for spaces for all required fields is done the same way.
		{"should forbid clientID when it only has spaces",
			"  ", testClientSecret, testSubscriptionID, testTenantID, testUserData, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.clientID")}))),
		},
		{"should forbid empty clientSecret",
			testClientID, "", testSubscriptionID, testTenantID, testUserData, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.clientSecret")}))),
		},
		{"should forbid empty subscriptionID",
			testClientID, testClientSecret, "", testTenantID, testUserData, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.subscriptionID")}))),
		},
		{"should forbid empty tenantID",
			testClientID, testClientSecret, testSubscriptionID, "", testUserData, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.tenantID")}))),
		},
		{
			"should forbid empty userData",
			testClientID, testClientSecret, testSubscriptionID, testTenantID, "", 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.userData")}))),
		},
		{"should forbid empty clientID and tenantID",
			"", testClientSecret, testSubscriptionID, "", testUserData, 2,
			ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.clientID")})),
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.tenantID")})),
			),
		},
		{"should forbid when all required fields are absent",
			"", "", "", "", "", 5,
			ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.clientID")})),
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.clientSecret")})),
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.subscriptionID")})),
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.tenantID")})),
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("data.userData")})),
			),
		},
		{"should succeed when all required fields are present", testClientID, testClientSecret, testSubscriptionID, testTenantID, testUserData, 0, nil},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			secret := createSecret(entry.clientID, entry.clientSecret, entry.subscriptionID, entry.tenantID, entry.testUserData)
			errList := ValidateProviderSecret(secret)
			g.Expect(len(errList)).To(Equal(entry.expectedErrors))
			if entry.matcher != nil {
				g.Expect(errList).To(entry.matcher)
			}
		})
	}
}

func TestValidateSubnetInfo(t *testing.T) {
	const (
		testSubnetName = "test-control-ns-nodes"
		testVnetName   = "test-control-ns"
	)

	fldPath := field.NewPath("providerSpec", "subnetInfo")

	table := []struct {
		description    string
		vnetName       string
		subnetName     string
		expectedErrors int
		matcher        gomegatypes.GomegaMatcher
	}{
		{"should forbid empty vnetName",
			"", testSubnetName, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.subnetInfo.vnetName")}))),
		},
		{"should forbid empty subnetName",
			testVnetName, "", 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.subnetInfo.subnetName")}))),
		},
		{"should forbid empty subnetName and vnetName",
			"", "", 2,
			ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.subnetInfo.vnetName")})),
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.subnetInfo.subnetName")})),
			),
		},
		{"should succeed when vnetName and subnetName are present", testVnetName, testSubnetName, 0, nil},
	}
	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			subnetInfo := api.AzureSubnetInfo{
				VnetName:   entry.vnetName,
				SubnetName: entry.subnetName,
			}
			errList := validateSubnetInfo(subnetInfo, fldPath)
			g.Expect(len(errList)).To(Equal(entry.expectedErrors))
			if entry.matcher != nil {
				g.Expect(errList).To(entry.matcher)
			}
		})
	}

}

func TestValidateHardwareProfile(t *testing.T) {
	fldPath := field.NewPath("providerSpec.properties.hardwareProfile")
	hwProfile := api.AzureHardwareProfile{}
	g := NewWithT(t)
	errList := validateHardwareProfile(hwProfile, fldPath)
	g.Expect(len(errList)).To(Equal(1))
	g.Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.hardwareProfile.vmSize")}))))
}

func TestValidateOSDisk(t *testing.T) {
	fldPath := field.NewPath("providerSpec.properties.storageProfile.osDisk")
	table := []struct {
		description    string
		osDisk         api.AzureOSDisk
		expectedErrors int
		matcher        gomegatypes.GomegaMatcher
	}{
		{
			"should forbid empty createOption",
			api.AzureOSDisk{Name: "osdisk-0", DiskSizeGB: 20, CreateOption: ""}, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.storageProfile.osDisk.createOption")}))),
		},
		{
			"should forbid zero osDisk size",
			api.AzureOSDisk{Name: "osdisk-0", DiskSizeGB: 0, CreateOption: "Create"}, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeInvalid), "Field": Equal("providerSpec.properties.storageProfile.osDisk.diskSizeGB")}))),
		},
		{
			"should forbid negative osDisk size",
			api.AzureOSDisk{Name: "osdisk-0", DiskSizeGB: -10, CreateOption: "Create"}, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeInvalid), "Field": Equal("providerSpec.properties.storageProfile.osDisk.diskSizeGB")}))),
		},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			errList := validateOSDisk(entry.osDisk, fldPath)
			g.Expect(len(errList)).To(Equal(entry.expectedErrors))
			if entry.matcher != nil {
				g.Expect(errList).To(entry.matcher)
			}
		})
	}
}

func TestValidateOSProfile(t *testing.T) {
	fldPath := field.NewPath("providerSpec.properties.osProfile")
	osProfile := api.AzureOSProfile{
		ComputerName:  "bingo",
		AdminUsername: "",
		LinuxConfiguration: api.AzureLinuxConfiguration{
			DisablePasswordAuthentication: true,
			SSH:                           api.AzureSSHConfiguration{},
		},
	}
	g := NewWithT(t)
	errList := validateOSProfile(osProfile, fldPath)
	g.Expect(len(errList)).To(Equal(1))
	g.Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.osProfile.adminUsername")}))))
}

func TestValidateDataDisks(t *testing.T) {
	fldPath := field.NewPath("providerSpec.properties.storageProfile.dataDisks")
	table := []struct {
		description    string
		disks          []api.AzureDataDisk
		expectedErrors int
		matcher        gomegatypes.GomegaMatcher
	}{
		{"should forbid empty storageAccountType",
			[]api.AzureDataDisk{{Name: "disk-1", Lun: pointer.Int32(0), StorageAccountType: "", DiskSizeGB: 10}}, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.storageProfile.dataDisks.storageAccountType")}))),
		},
		{"should forbid negative diskSize and empty storageAccountType",
			[]api.AzureDataDisk{{Name: "disk-1", Lun: pointer.Int32(0), StorageAccountType: "", DiskSizeGB: -10}}, 2,
			ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.storageProfile.dataDisks.storageAccountType")})),
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeInvalid), "Field": Equal("providerSpec.properties.storageProfile.dataDisks.diskSizeGB")})),
			),
		},
		{"should forbid duplicate Lun",
			[]api.AzureDataDisk{
				{Name: "disk-1", Lun: pointer.Int32(0), StorageAccountType: "StandardSSD_LRS", DiskSizeGB: 10},
				{Name: "disk-2", Lun: pointer.Int32(1), StorageAccountType: "StandardSSD_LRS", DiskSizeGB: 10},
				{Name: "disk-3", Lun: pointer.Int32(0), StorageAccountType: "StandardSSD_LRS", DiskSizeGB: 10},
				{Name: "disk-4", Lun: pointer.Int32(2), StorageAccountType: "StandardSSD_LRS", DiskSizeGB: 10},
				{Name: "disk-5", Lun: pointer.Int32(1), StorageAccountType: "StandardSSD_LRS", DiskSizeGB: 10},
			}, 2,
			ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeInvalid), "Field": Equal("providerSpec.properties.storageProfile.dataDisks.lun")})),
				PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeInvalid), "Field": Equal("providerSpec.properties.storageProfile.dataDisks.lun")})),
			),
		},
		{"should succeed with non-duplicate lun, valid diskSize and non-empty storageAccountType",
			[]api.AzureDataDisk{
				{Name: "disk-1", Lun: pointer.Int32(0), StorageAccountType: "StandardSSD_LRS", DiskSizeGB: 10},
				{Name: "disk-2", Lun: pointer.Int32(1), StorageAccountType: "StandardSSD_LRS", DiskSizeGB: 30},
				{Name: "disk-3", Lun: pointer.Int32(2), StorageAccountType: "StandardSSD_LRS", DiskSizeGB: 50},
			}, 0, nil,
		},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			errList := validateDataDisks(entry.disks, fldPath)
			g.Expect(len(errList)).To(Equal(entry.expectedErrors))
			if entry.matcher != nil {
				g.Expect(errList).To(entry.matcher)
			}
		})
	}
}

func TestValidateAvailabilityAndScalingConfig(t *testing.T) {
	var (
		testAvailabilitySet = api.AzureSubResource{ID: "availability-set-1"}
		testVMScaleSet      = api.AzureSubResource{ID: "vm-scale-set-1"}
	)
	fldPath := field.NewPath("providerSpec.properties")

	table := []struct {
		description     string
		zone            *int
		availabilitySet *api.AzureSubResource
		vmScaleSet      *api.AzureSubResource
		expectedErrors  int
		matcher         gomegatypes.GomegaMatcher
	}{
		{"should forbid zone, availabilitySet and virtualMachineScaleSet all to be set",
			pointer.Int(1), &testAvailabilitySet, &testVMScaleSet, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.zone|.availabilitySet|.virtualMachineScaleSet")}))),
		},
		{"should forbid setting availabilitySet when zone is set",
			pointer.Int(1), &testAvailabilitySet, nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.zone|.availabilitySet|.virtualMachineScaleSet")}))),
		},
		{"should forbid setting virtualMachineScaleSet when zone is set",
			pointer.Int(1), nil, &testVMScaleSet, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.zone|.availabilitySet|.virtualMachineScaleSet")}))),
		},
		{"should forbid setting both virtualMachineScaleSet and availabilitySet when zone is not set",
			nil, &testAvailabilitySet, &testVMScaleSet, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.zone|.availabilitySet|.virtualMachineScaleSet")}))),
		},
		{"should allow only setting of availabilitySet", nil, &testAvailabilitySet, nil, 0, nil},
		{"should allow only setting of zone", pointer.Int(1), nil, nil, 0, nil},
		{"should allow only setting of virtualMachineScaleSet", nil, nil, &testVMScaleSet, 0, nil},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			vmProperties := api.AzureVirtualMachineProperties{
				AvailabilitySet:        entry.availabilitySet,
				Zone:                   entry.zone,
				VirtualMachineScaleSet: entry.vmScaleSet,
			}
			errList := validateAvailabilityAndScalingConfig(vmProperties, fldPath)
			g.Expect(len(errList)).To(Equal(entry.expectedErrors))
			if entry.matcher != nil {
				g.Expect(errList).To(entry.matcher)
			}
		})
	}
}

func TestValidateStorageImageRef(t *testing.T) {
	const (
		testImageID                 = "storage-image-ID-test-1"
		testURN                     = "sap:gardenlinux:greatest:934.8.0"
		testSharedGalleryImageID    = "shared-gallery-image-ID-test-1"
		testCommunityGalleryImageID = "community-gallery-image-ID-test-1"
	)

	fldPath := field.NewPath("providerSpec.properties.storageProfile.imageReference")

	table := []struct {
		description             string
		id                      string
		urn                     *string
		sharedGalleryImageID    *string
		communityGalleryImageID *string
		expectedErrors          int
		matcher                 gomegatypes.GomegaMatcher
	}{
		{"should forbid setting of id, urn, communityGalleryImageID and sharedGalleryImageID",
			testImageID, pointer.String(testURN), pointer.String(testSharedGalleryImageID), pointer.String(testCommunityGalleryImageID), 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.storageProfile.imageReference.id|.urn|.communityGalleryImageID|.sharedGalleryImageID")}))),
		},
		{"should forbid setting of urn and id",
			testImageID, pointer.String(testURN), nil, nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.storageProfile.imageReference.id|.urn|.communityGalleryImageID|.sharedGalleryImageID")}))),
		},
		{"should forbid setting of communityGalleryImageID and id",
			testImageID, nil, nil, pointer.String(testCommunityGalleryImageID), 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.storageProfile.imageReference.id|.urn|.communityGalleryImageID|.sharedGalleryImageID")}))),
		},
		{"should forbid setting of sharedGalleryImageID and id",
			testImageID, nil, pointer.String(testSharedGalleryImageID), nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.storageProfile.imageReference.id|.urn|.communityGalleryImageID|.sharedGalleryImageID")}))),
		},
		{"should forbid setting of id, urn and communityGalleryImageID",
			testImageID, pointer.String(testURN), nil, pointer.String(testCommunityGalleryImageID), 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.storageProfile.imageReference.id|.urn|.communityGalleryImageID|.sharedGalleryImageID")}))),
		},
		{"should forbid setting of id, urn and sharedGalleryImageID",
			testImageID, pointer.String(testURN), pointer.String(testSharedGalleryImageID), nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.storageProfile.imageReference.id|.urn|.communityGalleryImageID|.sharedGalleryImageID")}))),
		},
		{"should forbid setting of communityGalleryImageID and sharedGalleryImageID",
			"", nil, pointer.String(testSharedGalleryImageID), pointer.String(testCommunityGalleryImageID), 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.storageProfile.imageReference.id|.urn|.communityGalleryImageID|.sharedGalleryImageID")}))),
		},
		{"should forbid setting of none of id, urn, communityGalleryImageID or sharedGalleryImageID",
			"", nil, nil, nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeForbidden), "Field": Equal("providerSpec.properties.storageProfile.imageReference.id|.urn|.communityGalleryImageID|.sharedGalleryImageID")}))),
		},
		{"should forbid invalid urn having less than 4 parts",
			"", pointer.String("sap:gardenlinux:greatest"), nil, nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeInvalid), "Field": Equal("providerSpec.properties.storageProfile.imageReference.urn")}))),
		},
		{"should forbid invalid urn with missing publisher",
			"", pointer.String(":gardenlinux:greatest:934.8.0"), nil, nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.storageProfile.imageReference.urn")}))),
		},
		{"should forbid invalid urn with missing offer",
			"", pointer.String("sap::greatest:934.8.0"), nil, nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.storageProfile.imageReference.urn")}))),
		},
		{"should forbid invalid urn with missing sku",
			"", pointer.String("sap:gardenlinux::934.8.0"), nil, nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.storageProfile.imageReference.urn")}))),
		},
		{"should forbid invalid urn with missing version",
			"", pointer.String("sap:gardenlinux:greatest:"), nil, nil, 1,
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.properties.storageProfile.imageReference.urn")}))),
		},
		{"should allow only id to be set", testImageID, nil, nil, nil, 0, nil},
		{"should allow only urn to be set", "", pointer.String(testURN), nil, nil, 0, nil},
		{"should allow only communityGalleryImageID to be set", "", nil, nil, pointer.String(testCommunityGalleryImageID), 0, nil},
		{"should allow only sharedGalleryImageID to be set", "", nil, pointer.String(testSharedGalleryImageID), nil, 0, nil},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			storageImageRef := api.AzureImageReference{
				ID:                      entry.id,
				URN:                     entry.urn,
				CommunityGalleryImageID: entry.communityGalleryImageID,
				SharedGalleryImageID:    entry.sharedGalleryImageID,
			}
			errList := validateStorageImageRef(storageImageRef, fldPath)
			g.Expect(len(errList)).To(Equal(entry.expectedErrors))
			if entry.matcher != nil {
				g.Expect(errList).To(entry.matcher)
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	fldPath := field.NewPath("providerSpec.tags")
	tags := map[string]string{
		"Name":                                             "shootns",
		"worker.gardener.cloud_pool":                       "worker-pool-0",
		"worker.garden.sapcloud.io_group":                  "worker-pool-0",
		"worker.gardener.cloud_cri-name":                   "containerd",
		"worker.gardener.cloud_system-components":          "true",
		"networking.gardener.cloud_node-local-dns-enabled": "true",
	}
	g := NewWithT(t)
	errList := validateTags(tags, fldPath)
	g.Expect(len(errList)).To(Equal(2))
	g.Expect(errList).To(ConsistOf(
		PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.tags.kubernetes.io-cluster-")})),
		PointTo(MatchFields(IgnoreExtras, Fields{"Type": Equal(field.ErrorTypeRequired), "Field": Equal("providerSpec.tags.kubernetes.io-role-")})),
	))
}

func createSecret(clientID, clientSecret, subscriptionID, tenantID, userData string) *corev1.Secret {
	data := make(map[string][]byte, 4)
	if !utils.IsEmptyString(clientID) {
		data["clientID"] = encodeAndConvertToBytes(clientID)
	}
	if !utils.IsEmptyString(clientSecret) {
		data["clientSecret"] = encodeAndConvertToBytes(clientSecret)
	}
	if !utils.IsEmptyString(subscriptionID) {
		data["subscriptionID"] = encodeAndConvertToBytes(subscriptionID)
	}
	if !utils.IsEmptyString(tenantID) {
		data["tenantID"] = encodeAndConvertToBytes(tenantID)
	}
	if !utils.IsEmptyString(userData) {
		data["userData"] = encodeAndConvertToBytes(userData)
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-secret",
		},
		Data: data,
		Type: "Opaque",
	}
}

func encodeAndConvertToBytes(value string) []byte {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(value)))
	base64.StdEncoding.Encode(dst, []byte(value))
	return dst
}
