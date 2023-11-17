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
