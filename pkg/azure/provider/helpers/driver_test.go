// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"fmt"
	"testing"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	. "github.com/onsi/gomega"
)

func TestDeriveInstanceID(t *testing.T) {
	const (
		vmName   = "vm-0"
		location = "eu-west-0"
	)
	expectedInstanceID := fmt.Sprintf("azure:///%s/%s", location, vmName)
	g := NewWithT(t)
	g.Expect(DeriveInstanceID(location, vmName)).To(Equal(expectedInstanceID))
}

func TestGetDiskNames(t *testing.T) {
	const (
		vmName                = "vm-0"
		testResourceGroupName = "test-rg"
		testShootNs           = "test-shoot-ns"
		testWorkerPool0Name   = "test-worker-pool-0"
		testDataDiskName      = "test-data-disk"
	)
	table := []struct {
		description       string
		numDataDisks      int
		expectedDiskCount int
	}{
		{"should return only 1 (OSDisk name) when there are no data disks", 0, 1},
		{"should return 3 disk names when there are 2 data disks set", 2, 3},
	}

	g := NewWithT(t)
	for _, entry := range table {
		// Setup
		// ---------------------------------------------------
		// create provider spec
		providerSpecBuilder := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues()
		if entry.numDataDisks > 0 {
			//Add data disks
			providerSpecBuilder.WithDataDisks(testDataDiskName, entry.numDataDisks)
		}
		providerSpec := providerSpecBuilder.Build()

		// Test
		// ------------------------------------------------
		actualDiskNames := GetDiskNames(providerSpec, vmName)
		g.Expect(actualDiskNames).To(HaveLen(entry.expectedDiskCount))
	}
}
