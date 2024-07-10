// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp/fakes"
)

const (
	testResourceGroupName = "test-rg"
	testShootNs           = "test-shoot-ns"
	testWorkerPool0Name   = "test-worker-pool-0"
)

func TestDecodeMachineSetConfig(t *testing.T) {
	vmssMachineSetConfig := api.AzureMachineSetConfig{ID: "vmss-0", Kind: api.MachineSetKindVMO}
	availabilitySetMachineSetConfig := api.AzureMachineSetConfig{ID: "as-0", Kind: api.MachineSetKindAvailabilitySet}
	table := []struct {
		description      string
		machineSetConfig api.AzureMachineSetConfig
		checkFn          func(g *WithT, decodedProviderSpec api.AzureProviderSpec)
	}{
		{
			"when deprecated MachineSetKindVMO is set, it should set VirtualMachineScaleSet property",
			vmssMachineSetConfig,
			func(g *WithT, decodedProviderSpec api.AzureProviderSpec) {
				g.Expect(decodedProviderSpec.Properties.VirtualMachineScaleSet).ToNot(BeNil())
				g.Expect(decodedProviderSpec.Properties.VirtualMachineScaleSet.ID).To(Equal(vmssMachineSetConfig.ID))
			},
		},
		{
			"when deprecated MachineSetKindAvailabilitySet is set, it should set AvailabilitySet property",
			availabilitySetMachineSetConfig,
			func(g *WithT, decodedProviderSpec api.AzureProviderSpec) {
				g.Expect(decodedProviderSpec.Properties.AvailabilitySet).ToNot(BeNil())
				g.Expect(decodedProviderSpec.Properties.AvailabilitySet.ID).To(Equal(availabilitySetMachineSetConfig.ID))
			},
		},
	}

	g := NewWithT(t)

	providerSpec := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).
		WithDefaultHardwareProfile().
		WithSubnetInfo("vnet-0").
		WithDefaultStorageProfile().
		WithDefaultOsProfile().
		WithDefaultTags().
		Build()
	providerSpec.Properties.Zone = nil

	for _, entry := range table {
		t.Run(entry.description, func(_ *testing.T) {
			providerSpec.Properties.MachineSet = &entry.machineSetConfig
			machineClass, err := fakes.CreateMachineClass(providerSpec, nil)
			g.Expect(err).To(BeNil())
			decodedProviderSpec, err := DecodeAndValidateMachineClassProviderSpec(machineClass)
			g.Expect(err).To(BeNil())
			entry.checkFn(g, decodedProviderSpec)
		})
	}
}
