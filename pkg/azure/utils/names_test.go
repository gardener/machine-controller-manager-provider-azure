// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"testing"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	. "github.com/onsi/gomega"
)

const vmName = "shoot--test-project-z1-4567c-xj5sq"

func TestCreateOSDiskName(t *testing.T) {
	g := NewWithT(t)
	g.Expect(CreateOSDiskName(vmName)).To(Equal(fmt.Sprintf("%s-os-disk", vmName)))
}

func TestCreateNICName(t *testing.T) {
	g := NewWithT(t)
	g.Expect(CreateNICName(vmName)).To(Equal(fmt.Sprintf("%s-nic", vmName)))
}

func TestCreateDataDiskName(t *testing.T) {
	table := []struct {
		description          string
		vmName               string
		dataDiskName         string
		lun                  int32
		expectedDataDiskName string
	}{
		{
			"should include vmName, data disk name and lun, when data disk name is not empty", vmName, "dd1", 1, fmt.Sprintf("%s-dd1-1-data-disk", vmName),
		},
		{
			"should exclude data disk name but include vmName and lun, when data disk name is empty", vmName, "", 1, fmt.Sprintf("%s-1-data-disk", vmName),
		},
	}
	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			dataDisk := api.AzureDataDisk{
				Name:       entry.dataDiskName,
				Lun:        entry.lun,
				DiskSizeGB: 10,
			}
			g.Expect(CreateDataDiskName(vmName, dataDisk.Name, dataDisk.Lun)).To(Equal(entry.expectedDataDiskName))
		})
	}
}

func TestExtractVMNameFromNICName(t *testing.T) {
	const nicName = "shoot--test-project-z1-4567c-xj5sq-nic"
	g := NewWithT(t)
	g.Expect(ExtractVMNameFromNICName(nicName)).To(Equal(vmName))
}

func TestExtractVMNameFromOSDiskName(t *testing.T) {
	const nicName = "shoot--test-project-z1-4567c-xj5sq-os-disk"
	g := NewWithT(t)
	g.Expect(ExtractVMNameFromOSDiskName(nicName)).To(Equal(vmName))
}
