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

func TestDetermineAzureCloudConfiguration(t *testing.T) {
	type testData struct {
		description       string
		testConfiguration *api.CloudConfiguration
		expectedOutput    *cloud.Configuration
	}

	tests := []testData{
		{description: "cloud configuration name set to AzurePublic", testConfiguration: &api.CloudConfiguration{Name: api.CloudNamePublic}, expectedOutput: &cloud.AzurePublic},
		{description: "cloud configuration name set to AzureChina", testConfiguration: &api.CloudConfiguration{Name: api.CloudNameChina}, expectedOutput: &cloud.AzureChina},
		{description: "cloud configuration name set to AzureGov", testConfiguration: &api.CloudConfiguration{Name: api.CloudNameGov}, expectedOutput: &cloud.AzureGovernment},
		{description: "cloud configuration not set", testConfiguration: nil, expectedOutput: &cloud.AzurePublic},
	}
	g := NewWithT(t)
	t.Parallel()
	for _, test := range tests {
		t.Run(test.description, func(_ *testing.T) {
			cloudConfiguration := DetermineAzureCloudConfiguration(test.testConfiguration)
			g.Expect(cloudConfiguration).To(Equal(*test.expectedOutput))
		})
	}
}
