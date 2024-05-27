// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	. "github.com/onsi/gomega"
)

func TestNilConfig(t *testing.T) {
	g := NewWithT(t)

	testSpec := &api.AzureProviderSpec{}

	configuration, err := ExtractCloudConfiguration(testSpec)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzurePublic))
}

func TestInvalidConfigName(t *testing.T) {
	g := NewWithT(t)

	testSpec := &api.AzureProviderSpec{
		CloudConfiguration: &api.CloudConfiguration{
			Name: "Foo",
		},
	}

	_, err := ExtractCloudConfiguration(testSpec)

	g.Expect(err).To(HaveOccurred())
}

func TestPredefinedClouds(t *testing.T) {
	g := NewWithT(t)

	testPublicConfiguration := &api.CloudConfiguration{
		Name: api.AzurePublicCloudName,
	}
	testGovConfiguration := &api.CloudConfiguration{
		Name: api.AzureGovCloudName,
	}
	testChinaConfigration := &api.CloudConfiguration{
		Name: api.AzureChinaCloudName,
	}
	testSpec := &api.AzureProviderSpec{}

	testSpec.CloudConfiguration = testPublicConfiguration

	configuration, err := ExtractCloudConfiguration(testSpec)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzurePublic))

	testSpec.CloudConfiguration = testGovConfiguration

	configuration, err = ExtractCloudConfiguration(testSpec)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzureGovernment))

	testSpec.CloudConfiguration = testChinaConfigration

	configuration, err = ExtractCloudConfiguration(testSpec)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(configuration).To(Equal(cloud.AzureChina))
}
