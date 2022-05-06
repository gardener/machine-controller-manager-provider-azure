// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package validation - validation is used to validate cloud specific ProviderSpec

package validation_test

import (
	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	. "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis/validation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

var _ = Describe("#Validate", func() {

	DescribeTable("#DescribeTable",
		func(imageRef api.AzureImageReference, matcher gomegatypes.GomegaMatcher) {
			fieldPath := field.NewPath("storageProfile.imageReference")
			Expect(ValidateImageReference(imageRef, fieldPath)).To(matcher)
		},
		Entry("should allow to specify urn",
			api.AzureImageReference{
				URN: pointer.StringPtr("abc:def:ghi:jkl"),
			},
			HaveLen(0),
		),
		Entry("should allow to specify community image id",
			api.AzureImageReference{
				CommunityGalleryImageID: pointer.StringPtr("test-community-image-id"),
			},
			HaveLen(0),
		),
		Entry("should allow to specify image id",
			api.AzureImageReference{
				ID: "test-image-id",
			},
			HaveLen(0),
		),
		Entry("should forbid as no urn, no community image and no image id is specified",
			api.AzureImageReference{},
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("storageProfile.imageReference"),
			}))),
		),
		Entry("should forbid to specify community image id or image id when an urn is specified",
			api.AzureImageReference{
				URN:                     pointer.StringPtr("abc:def:ghi:jkl"),
				CommunityGalleryImageID: pointer.StringPtr("test-community-image-id"),
				ID:                      "test-image-id",
			},
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("storageProfile.imageReference.urn"),
			}))),
		),
		Entry("should forbid to specify an invalid urn",
			api.AzureImageReference{
				URN: pointer.StringPtr("abc.def.ghi"),
			},
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("storageProfile.imageReference.urn"),
			}))),
		),
		Entry("should forbid to specify an urn with an invalid part",
			api.AzureImageReference{
				URN: pointer.StringPtr("abc:def::jkl"),
			},
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("storageProfile.imageReference.urn"),
			}))),
		),
		Entry("should forbid to specify a community image id and an image id",
			api.AzureImageReference{
				CommunityGalleryImageID: pointer.StringPtr("test-community-image-id"),
				ID:                      "test-image-id",
			},
			ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("storageProfile.imageReference.communityGalleryImageID"),
			}))),
		),
	)

})
