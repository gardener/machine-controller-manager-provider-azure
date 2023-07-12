package provider

import (
	"context"
	"encoding/json"
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

func TestDeleteMachine(t *testing.T) {
	table := []struct {
		description          string
		existingVMNames      []string
		targetVMNameToDelete string
		providerSpec         api.AzureProviderSpec
		cascadeDeleteOpts    fakes.CascadeDeleteOpts
		checkClusterStateFn  func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string)
	}{
		{
			"should delete all resources when cascade delete is set for NIC and all disks",
			[]string{"vm-0", "vm-1"},
			"vm-1",
			test.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().Build(),
			fakes.CascadeDeleteAllResources,
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
			[]string{"vm-0", "vm-1"},
			"vm-1",
			test.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().Build(),
			fakes.CascadeDeleteOpts{
				OSDisk:   to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
				DataDisk: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
			},
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
	}

	g := NewWithT(t)
	ctx := context.TODO()

	for _, entry := range table {
		t.Log(entry.description)
		clusterState := fakes.NewClusterState(testResourceGroupName)
		// initialize cluster state
		for _, vmName := range entry.existingVMNames {
			clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(entry.providerSpec, vmName).WithCascadeDeleteOptions(entry.cascadeDeleteOpts).Build())
		}
		fakeFactory := createDefaultFakeFactoryForDeleteAPI(g, testResourceGroupName, clusterState)
		machineClass, err := createMachineClass(entry.providerSpec)
		g.Expect(err).To(BeNil())

		// Test environment before running actual test
		//----------------------------------------------------------------------------
		_, err = fakeFactory.VMAccess.Get(ctx, testResourceGroupName, entry.targetVMNameToDelete, nil)
		g.Expect(err).To(BeNil())

		machine := &v1alpha1.Machine{
			ObjectMeta: newMachineObjectMeta(testShootNs, entry.targetVMNameToDelete),
		}

		// Test
		//----------------------------------------------------------------------------
		testDriver := NewDefaultDriver(fakeFactory)
		_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
			Machine:      machine,
			MachineClass: machineClass,
			Secret:       test.CreateProviderSecret(),
		})
		g.Expect(err).To(BeNil())
		entry.checkClusterStateFn(g, ctx, *fakeFactory, entry.targetVMNameToDelete)
	}
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

func createMachineClass(providerSpec api.AzureProviderSpec) (*v1alpha1.MachineClass, error) {
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
