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

func TestCloudConfigurationDetermination(t *testing.T) {
	g := NewWithT(t)

	type testData struct {
		testConfiguration *api.CloudConfiguration
		expectedOutput    *cloud.Configuration
		shouldFail        bool
	}

	tests := []testData{
		{testConfiguration: &api.CloudConfiguration{Name: api.CloudNamePublic}, expectedOutput: &cloud.AzurePublic},
		{testConfiguration: &api.CloudConfiguration{Name: api.CloudNameChina}, expectedOutput: &cloud.AzureChina},
		{testConfiguration: &api.CloudConfiguration{Name: api.CloudNameGov}, expectedOutput: &cloud.AzureGovernment},
		{testConfiguration: &api.CloudConfiguration{Name: "FooCloud"}, expectedOutput: nil, shouldFail: true},
		{testConfiguration: nil, expectedOutput: &cloud.AzurePublic},
	}

	for _, t := range tests {
		cloudConfiguration, err := DetermineCloudConfiguration(t.testConfiguration)
		if t.shouldFail {
			g.Expect(err).To(HaveOccurred())
		}
		if t.expectedOutput != nil {
			g.Expect(cloudConfiguration).To(Equal(*t.expectedOutput))
		}

	}

}
