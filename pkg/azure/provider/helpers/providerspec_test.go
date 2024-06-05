// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

func TestNilConfig(t *testing.T) {
	g := NewWithT(t)

	var (
		testConfig *api.CloudConfiguration = nil
		testRegion                         = ptr.To("Foo")
	)

	configuration, err := DetermineCloudConfiguration(testConfig, testRegion)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzurePublic))
}

func TestNilRegion(t *testing.T) {
	g := NewWithT(t)

	var (
		testConfig         = &api.CloudConfiguration{Name: api.AzurePublicCloudName}
		testRegion *string = nil
	)

	configuration, err := DetermineCloudConfiguration(testConfig, testRegion)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzurePublic))
}

func TestNilConfigAndRegion(t *testing.T) {
	g := NewWithT(t)

	var (
		testConfig *api.CloudConfiguration = nil
		testRegion *string                 = nil
	)

	configuration, err := DetermineCloudConfiguration(testConfig, testRegion)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzurePublic))
}

func TestInvalidConfigName(t *testing.T) {
	g := NewWithT(t)

	var (
		testConfig         = &api.CloudConfiguration{Name: "Foo"}
		testRegion *string = nil
	)

	_, err := DetermineCloudConfiguration(testConfig, testRegion)

	g.Expect(err).To(HaveOccurred())
}

func TestPredefinedClouds(t *testing.T) {
	g := NewWithT(t)

	var (
		testPublicConfiguration         = &api.CloudConfiguration{Name: api.AzurePublicCloudName}
		testGovConfiguration            = &api.CloudConfiguration{Name: api.AzureGovCloudName}
		testChinaConfigration           = &api.CloudConfiguration{Name: api.AzureChinaCloudName}
		testRegion              *string = nil
	)

	configuration, err := DetermineCloudConfiguration(testPublicConfiguration, testRegion)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzurePublic))

	configuration, err = DetermineCloudConfiguration(testGovConfiguration, testRegion)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzureGovernment))

	configuration, err = DetermineCloudConfiguration(testChinaConfigration, testRegion)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzureChina))
}

func TestRegionMatching(t *testing.T) {
	g := NewWithT(t)

	var (
		testConfig *api.CloudConfiguration = nil
		testRegion *string                 = ptr.To("ussecFoo")
	)

	configuration, err := DetermineCloudConfiguration(testConfig, testRegion)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzureGovernment))

}
