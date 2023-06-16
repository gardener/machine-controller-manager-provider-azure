package helpers

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	. "github.com/onsi/gomega"
)

const (
	testResourceGroup = "shoot--mb-garden--sdktest"
	testVMName        = "shoot--mb-garden--sdktest-worker-bingo"
	testLocation      = "westeurope"
	testPlanName      = "greatest"
	testProduct       = "gardenlinux"
	testPublisher     = "sap"
	testAdminUserName = "core"
	testZone          = "1"
)

func TestIsVMCascadeDeleteSetForNICs(t *testing.T) {
	table := []struct {
		description    string
		vm             *armcompute.VirtualMachine
		expectedResult bool
	}{
		{"should return false if vm is nil", nil, false},
		{"should return false if vm.properties is nil", newVmSpecGenerator().GetVM(), false},
		{"should return false if NetworkProfile is nil", newVmSpecGenerator().WithEmptyProperties().GetVM(), false},
		{"should return false if NetworkProfile is empty", newVmSpecGenerator().WithEmptyNetworkInterfaceReferences().GetVM(), false},
		{"should return false if none of the NetworkInterfaces have DeleteOption set",
			newVmSpecGenerator().
				WithNIC("nic-1", nil).
				WithNIC("nic-2", nil).GetVM(), false,
		},
		{"should return false if one of NetworkInterfaces has no DeleteOption set",
			newVmSpecGenerator().
				WithNIC("nic-1", to.Ptr(armcompute.DeleteOptionsDelete)).
				WithNIC("nic-2", nil).GetVM(), false,
		},
		{"should return false if one of the NetworkInterfaces have DeleteOption set to Detach",
			newVmSpecGenerator().
				WithNIC("nic-1", to.Ptr(armcompute.DeleteOptionsDetach)).
				WithNIC("nic-2", to.Ptr(armcompute.DeleteOptionsDelete)).GetVM(), false,
		},
		{"should return true if all of the NetworkInterfaces have DeleteOption set to Delete",
			newVmSpecGenerator().
				WithNIC("nic-1", to.Ptr(armcompute.DeleteOptionsDelete)).
				WithNIC("nic-2", to.Ptr(armcompute.DeleteOptionsDelete)).GetVM(), true,
		},
	}
	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry.description)
		actualResult := IsVMCascadeDeleteSetForNICs(entry.vm)
		g.Expect(actualResult).To(Equal(entry.expectedResult))
	}
}

// ---------------------------------------------------- Utility functions for unit tests -----------------------------------------------------

func newVmSpecGenerator() *vmSpecGenerator {
	generator := vmSpecGenerator{}
	generator.generateTestVMSpec()
	return &generator
}

type vmSpecGenerator struct {
	vm *armcompute.VirtualMachine
}

func (g *vmSpecGenerator) generateTestVMSpec() *vmSpecGenerator {
	tags := make(map[string]*string)
	tags["name"] = to.Ptr(testVMName)

	vm := &armcompute.VirtualMachine{
		Location: to.Ptr(testLocation),
		Plan: &armcompute.Plan{
			Name:      to.Ptr(testPlanName),
			Product:   to.Ptr(testProduct),
			Publisher: to.Ptr(testPublisher),
		},
		Tags:  tags,
		Zones: []*string{to.Ptr(testZone)},
		Name:  to.Ptr(testVMName),
	}
	g.vm = vm
	return g
}

func (g *vmSpecGenerator) WithEmptyProperties() *vmSpecGenerator {
	g.vm.Properties = &armcompute.VirtualMachineProperties{}
	return g
}

func (g *vmSpecGenerator) WithEmptyNetworkProfile() *vmSpecGenerator {
	if g.vm.Properties == nil {
		g.WithEmptyProperties()
	}
	g.vm.Properties.NetworkProfile = &armcompute.NetworkProfile{}
	return g
}

func (g *vmSpecGenerator) WithEmptyNetworkInterfaceReferences() *vmSpecGenerator {
	if g.vm.Properties == nil {
		g.WithEmptyProperties()
	}
	if g.vm.Properties.NetworkProfile == nil {
		g.WithEmptyNetworkProfile()
	}
	g.vm.Properties.NetworkProfile.NetworkInterfaces = []*armcompute.NetworkInterfaceReference{}
	return g
}

func (g *vmSpecGenerator) WithNIC(nicName string, deleteOption *armcompute.DeleteOptions) *vmSpecGenerator {
	if g.vm.Properties == nil {
		g.WithEmptyProperties()
	}
	if g.vm.Properties.NetworkProfile == nil {
		g.WithEmptyNetworkProfile()
	}
	if g.vm.Properties.NetworkProfile.NetworkInterfaces == nil {
		g.WithEmptyNetworkInterfaceReferences()
	}
	g.vm.Properties.NetworkProfile.NetworkInterfaces = append(g.vm.Properties.NetworkProfile.NetworkInterfaces, &armcompute.NetworkInterfaceReference{
		ID: to.Ptr(nicName),
		Properties: &armcompute.NetworkInterfaceReferenceProperties{
			DeleteOption: deleteOption,
		},
	})
	return g
}

func (g *vmSpecGenerator) GetVM() *armcompute.VirtualMachine {
	return g.vm
}
