package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test/fakes"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
)

const (
	testResourceGroupName = "test-rg"
	testShootNs           = "test-shoot-ns"
	testWorkerPool0Name   = "test-worker-pool-0"
)

func TestDeleteMachineWhenVMExists(t *testing.T) {
	table := []struct {
		description                string
		resourceGroup              string                  // used to build initial ClusterState
		existingVMNames            []string                // used to build initial ClusterState
		numDataDisks               int                     // used to build initial ClusterState
		cascadeDeleteOpts          fakes.CascadeDeleteOpts // used to build initial ClusterState
		machineClassResourceGroup  *string                 // for tests where a different resource Group than used to create ClusterState needs to be passed.
		targetVMNameToDelete       string                  // name of the VM that will be deleted via DeleteMachine
		shouldDeleteMachineSucceed bool
		checkClusterStateFn        func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string)
	}{
		{
			"should delete all resources(NIC and OSDisk) when cascade delete is set for NIC and all Disks",
			testResourceGroupName,
			[]string{"vm-0", "vm-1"},
			1,
			fakes.CascadeDeleteAllResources,
			nil,
			"vm-1",
			true,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string) {
				_, err := factory.VMAccess.Get(ctx, testResourceGroupName, vmName, nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
				_, err = factory.InterfaceAccess.Get(ctx, testResourceGroupName, utils.CreateNICName(vmName), nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
				_, err = factory.DisksAccess.Get(ctx, testResourceGroupName, utils.CreateOSDiskName(vmName), nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
			},
		},
		{
			"should update VM before deleting the VM when cascade delete is not set for NIC",
			testResourceGroupName,
			[]string{"vm-0", "vm-1"},
			2,
			fakes.CascadeDeleteOpts{
				OSDisk:   to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
				DataDisk: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
			},
			nil,
			"vm-0",
			true,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string) {
				_, err := factory.VMAccess.Get(ctx, testResourceGroupName, vmName, nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
				_, err = factory.DisksAccess.Get(ctx, testResourceGroupName, utils.CreateOSDiskName(vmName), nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
				_, err = factory.InterfaceAccess.Get(ctx, testResourceGroupName, utils.CreateNICName(vmName), nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
			},
		},
		{
			"should update VM before deleting the VM when cascade delete is not set for NIC and Disks",
			testResourceGroupName,
			[]string{"vm-0", "vm-1"},
			0,
			fakes.CascadeDeleteOpts{},
			nil,
			"vm-1",
			true,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string) {
				_, err := factory.VMAccess.Get(ctx, testResourceGroupName, vmName, nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
				_, err = factory.DisksAccess.Get(ctx, testResourceGroupName, utils.CreateOSDiskName(vmName), nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
				_, err = factory.InterfaceAccess.Get(ctx, testResourceGroupName, utils.CreateNICName(vmName), nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
			},
		},
		{
			"should skip delete if the resource group is not found",
			testResourceGroupName,
			[]string{"vm-0", "vm-1"},
			0,
			fakes.CascadeDeleteOpts{},
			to.Ptr("wrong-resource-group"),
			"vm-1",
			true,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string) {
				vmResp, err := factory.VMAccess.Get(ctx, testResourceGroupName, vmName, nil)
				g.Expect(err).To(BeNil())
				g.Expect(*vmResp.VirtualMachine.Name).To(Equal(vmName))
				osDiskResp, err := factory.DisksAccess.Get(ctx, testResourceGroupName, utils.CreateOSDiskName(vmName), nil)
				g.Expect(err).To(BeNil())
				g.Expect(osDiskResp.ManagedBy).ToNot(BeNil())
				nicResp, err := factory.InterfaceAccess.Get(ctx, testResourceGroupName, utils.CreateNICName(vmName), nil)
				g.Expect(err).To(BeNil())
				g.Expect(nicResp.Interface.Properties.VirtualMachine).ToNot(BeNil())
			},
		},
	}

	g := NewWithT(t)
	ctx := context.TODO()

	for _, entry := range table {
		t.Log(entry.description)
		// initialize cluster state
		//----------------------------------------------------------------------------

		// create provider spec
		providerSpecBuilder := test.NewProviderSpecBuilder(entry.resourceGroup, testShootNs, testWorkerPool0Name).WithDefaultValues()
		if entry.numDataDisks > 0 {
			//Add data disks
			providerSpecBuilder.WithDataDisks(createDataDiskNames(entry.numDataDisks))
		}
		providerSpec := providerSpecBuilder.Build()

		// create cluster state
		clusterState := fakes.NewClusterState(providerSpec.ResourceGroup)
		for _, vmName := range entry.existingVMNames {
			clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).WithCascadeDeleteOptions(entry.cascadeDeleteOpts).Build())
		}
		// create fake factory
		fakeFactory := createDefaultFakeFactoryForDeleteAPI(g, providerSpec.ResourceGroup, clusterState)

		// Create machine and machine class to be used to create DeleteMachineRequest
		machineClass, err := createMachineClass(providerSpec, entry.machineClassResourceGroup)
		g.Expect(err).To(BeNil())
		machine := &v1alpha1.Machine{
			ObjectMeta: newMachineObjectMeta(testShootNs, entry.targetVMNameToDelete),
		}

		// Test environment before running actual test
		//----------------------------------------------------------------------------
		_, err = fakeFactory.VMAccess.Get(ctx, providerSpec.ResourceGroup, entry.targetVMNameToDelete, nil)
		g.Expect(err).To(BeNil())

		// Test
		//----------------------------------------------------------------------------
		testDriver := NewDefaultDriver(fakeFactory)
		_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
			Machine:      machine,
			MachineClass: machineClass,
			Secret:       test.CreateProviderSecret(),
		})
		g.Expect(err == nil).To(Equal(entry.shouldDeleteMachineSucceed))

		// evaluate cluster state post delete machine operation
		entry.checkClusterStateFn(g, ctx, *fakeFactory, entry.targetVMNameToDelete)
	}
}

func TestDeleteMachineWhenVMDoesNotExist(t *testing.T) {

}

func TestDeleteMachineWhenProviderIsNotAzure(t *testing.T) {

}

func createDefaultFakeFactoryForDeleteAPI(g *WithT, resourceGroup string, clusterState *fakes.ClusterState) *fakes.Factory {
	fakeFactory := fakes.NewFactory(resourceGroup)
	vmAccess, err := fakeFactory.NewVirtualMachineAccessBuilder().WithClusterState(clusterState).WithDefaultAPIBehavior().Build()
	g.Expect(err).To(BeNil())
	nicAccess, err := fakeFactory.NewNICAccessBuilder().WithClusterState(clusterState).WithDefaultAPIBehavior().Build()
	g.Expect(err).To(BeNil())
	rgAccess, err := fakeFactory.NewResourceGroupsAccessBuilder().WithCheckExistence(nil).Build()
	g.Expect(err).To(BeNil())
	diskAccess, err := fakeFactory.NewDiskAccessBuilder().WithClusterState(clusterState).WithDefaultAPIBehavior().Build()
	g.Expect(err).To(BeNil())
	fakeFactory.
		WithVirtualMachineAccess(vmAccess).
		WithResourceGroupsAccess(rgAccess).
		WithNetworkInterfacesAccess(nicAccess).
		WithDisksAccess(diskAccess)
	return fakeFactory
}

func createMachineClass(providerSpec api.AzureProviderSpec, resourceGroup *string) (*v1alpha1.MachineClass, error) {
	if resourceGroup != nil {
		providerSpec.ResourceGroup = *resourceGroup
	}
	specBytes, err := json.Marshal(providerSpec)
	if err != nil {
		return nil, err
	}
	machineClass := &v1alpha1.MachineClass{
		Provider: "Azure",
		ProviderSpec: runtime.RawExtension{
			Raw:    specBytes,
			Object: nil,
		},
	}
	return machineClass, nil
}

func newMachineObjectMeta(namespace string, vmName string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: namespace,
		Name:      vmName,
	}
}

func createDataDiskNames(numDataDisks int) []string {
	diskNames := make([]string, 0, numDataDisks)
	for i := 0; i < numDataDisks; i++ {
		diskName := fmt.Sprintf("test-disk-%d", i)
		diskNames = append(diskNames, diskName)
	}
	return diskNames
}
