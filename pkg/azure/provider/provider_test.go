package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	accesserrors "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/provider/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp/fakes"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	. "github.com/onsi/gomega"
)

const (
	testResourceGroupName = "test-rg"
	testShootNs           = "test-shoot-ns"
	testWorkerPool0Name   = "test-worker-pool-0"
	testDataDiskName      = "test-data-disk"
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
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false)
			},
		},
		{
			"should update VM before deleting the VM when cascade delete is not set for NIC but its set for disks",
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
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false)
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
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false)
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
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, true, true, true)
			},
		},
	}

	g := NewWithT(t)
	ctx := context.Background()

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// initialize cluster state
			//----------------------------------------------------------------------------
			// create provider spec
			providerSpecBuilder := testhelp.NewProviderSpecBuilder(entry.resourceGroup, testShootNs, testWorkerPool0Name).WithDefaultValues()
			if entry.numDataDisks > 0 {
				//Add data disks
				providerSpecBuilder.WithDataDisks(testDataDiskName, entry.numDataDisks)
			}
			providerSpec := providerSpecBuilder.Build()

			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec.ResourceGroup)
			for _, vmName := range entry.existingVMNames {
				clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).WithCascadeDeleteOptions(entry.cascadeDeleteOpts).BuildAllResources())
			}
			// create fake factory
			fakeFactory := createDefaultFakeFactoryForMachineDelete(g, providerSpec.ResourceGroup, clusterState)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := testhelp.CreateMachineClass(providerSpec, entry.machineClassResourceGroup)
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: testhelp.NewMachineObjectMeta(testShootNs, entry.targetVMNameToDelete),
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
				Secret:       testhelp.CreateProviderSecret(),
			})
			g.Expect(err == nil).To(Equal(entry.shouldDeleteMachineSucceed))

			// evaluate cluster state post delete machine operation
			entry.checkClusterStateFn(g, ctx, *fakeFactory, entry.targetVMNameToDelete)
		})
	}
}

func TestDeleteMachineWhenVMDoesNotExist(t *testing.T) {
	const vmName = "test-vm-0"
	testVMID := testhelp.CreateVirtualMachineID(testhelp.SubscriptionID, testResourceGroupName, vmName)

	table := []struct {
		description                string
		nicPresent                 bool
		osDiskPresent              bool
		numDataDisks               int
		vmID                       *string
		shouldDeleteMachineSucceed bool
		checkClusterStateFn        func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string)
	}{
		{
			"should delete left over NIC and Disks when they are detached from VM",
			true, true, 1, nil, true,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				_, err := factory.DisksAccess.Get(ctx, testResourceGroupName, utils.CreateOSDiskName(vmName), nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
				for _, dataDiskName := range dataDiskNames {
					_, err := factory.DisksAccess.Get(ctx, testResourceGroupName, dataDiskName, nil)
					g.Expect(err).ToNot(BeNil())
					g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
				}
				_, err = factory.InterfaceAccess.Get(ctx, testResourceGroupName, utils.CreateNICName(vmName), nil)
				g.Expect(err).ToNot(BeNil())
				g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
			},
		},
		{
			"should fail delete of NIC when its still associated with a VM",
			true, false, 0, &testVMID, false,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				nic, err := factory.InterfaceAccess.Get(ctx, testResourceGroupName, utils.CreateNICName(vmName), nil)
				g.Expect(err).To(BeNil())
				g.Expect(nic.Properties.VirtualMachine).ToNot(BeNil())
				g.Expect(*nic.Properties.VirtualMachine.ID).To(Equal(testVMID))
			},
		},
		{
			"should fail delete of disks when its still associated with a VM",
			false, true, 1, &testVMID, false,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				osDisk, err := factory.DisksAccess.Get(ctx, testResourceGroupName, utils.CreateOSDiskName(vmName), nil)
				g.Expect(err).To(BeNil())
				g.Expect(osDisk.ManagedBy).ToNot(BeNil())
				g.Expect(*osDisk.ManagedBy).To(Equal(testVMID))
				for _, dataDiskName := range dataDiskNames {
					dataDisk, err := factory.DisksAccess.Get(ctx, testResourceGroupName, dataDiskName, nil)
					g.Expect(err).To(BeNil())
					g.Expect(*dataDisk.ManagedBy).ToNot(BeNil())
					g.Expect(*dataDisk.ManagedBy).To(Equal(testVMID))
				}
			},
		},
	}

	g := NewWithT(t)
	ctx := context.Background()

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// initialize cluster state
			//----------------------------------------------------------------------------
			// create provider spec
			providerSpecBuilder := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues()
			if entry.numDataDisks > 0 {
				//Add data disks
				providerSpecBuilder.WithDataDisks(testDataDiskName, entry.numDataDisks)
			}
			providerSpec := providerSpecBuilder.Build()

			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec.ResourceGroup)
			clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).BuildWith(false, entry.nicPresent, entry.osDiskPresent, entry.numDataDisks > 0, entry.vmID))

			// create fake factory
			fakeFactory := createDefaultFakeFactoryForMachineDelete(g, providerSpec.ResourceGroup, clusterState)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := testhelp.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: testhelp.NewMachineObjectMeta(testShootNs, vmName),
			}

			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       testhelp.CreateProviderSecret(),
			})
			g.Expect(err == nil).To(Equal(entry.shouldDeleteMachineSucceed))

			dataDiskNames := testhelp.CreateDataDiskNames(vmName, providerSpec)
			entry.checkClusterStateFn(g, ctx, *fakeFactory, vmName, dataDiskNames)
		})
	}
}

func TestDeleteMachineWithInducedErrors(t *testing.T) {
	const (
		testErrorCode = "test-error-code"
		vmName        = "test-vm-0"
	)
	testInternalServerError := fakes.InternalServerError(testErrorCode)
	table := []struct {
		description               string
		vmAccessAPIBehaviorSpec   *fakes.APIBehaviorSpec
		rgAccessAPIBehaviorSpec   *fakes.APIBehaviorSpec
		diskAccessAPIBehaviorSpec *fakes.APIBehaviorSpec
		nicAccessAPIBehaviorSpec  *fakes.APIBehaviorSpec
		cascadeDeleteOpts         fakes.CascadeDeleteOpts
		vmExists                  bool
		checkErrorFn              func(g *WithT, err error)
		checkClusterStateFn       func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string)
	}{
		{
			"should fail when checking resource groups existence returns an error", nil,
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(testResourceGroupName, testhelp.AccessMethodCheckExistence, testInternalServerError),
			nil, nil, fakes.CascadeDeleteAllResources, true, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				createFactoryAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, true, true, true)
			},
		},
		{
			"should fail when VM access Get call returns an error",
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(vmName, testhelp.AccessMethodGet, testInternalServerError),
			nil, nil, nil, fakes.CascadeDeleteAllResources, true, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				createFactoryAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, true, true, true)
			},
		},
		{
			"should delete left over OSDisk but fail to delete left over NIC when VM does not exist",
			nil, nil, nil,
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(utils.CreateNICName(vmName), testhelp.AccessMethodBeginDelete, testInternalServerError), fakes.CascadeDeleteAllResources, false, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				createFactoryAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, false, true, false)
			},
		},
		{
			"should fail when existing VM's cascade delete options update returns an error",
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(vmName, testhelp.AccessMethodBeginUpdate, testInternalServerError),
			nil, nil, nil, fakes.CascadeDeleteOpts{}, true, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				createFactoryAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, true, true, true)
			},
		},
		{
			"should fail when deletion of the VM post update of cascade deletion option completely fails",
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(vmName, testhelp.AccessMethodBeginDelete, testInternalServerError),
			nil, nil, nil, fakes.CascadeDeleteOpts{}, true, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				factory := createDefaultFakeFactoryForMachineDelete(g, testResourceGroupName, clusterState)
				machineResources := checkClusterStateAndGetMachineResources(g, ctx, *factory, vmName, true, true, true)
				// validate that the cascade delete options are now set
				g.Expect(machineResources.VM).ToNot(BeNil())
				checkCascadeDeleteOptions(t, *machineResources.VM, fakes.CascadeDeleteAllResources)
			},
		},
	}

	g := NewWithT(t)
	ctx := context.Background()
	// create provider spec
	providerSpec := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().Build()

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// initialize cluster state
			//----------------------------------------------------------------------------
			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec.ResourceGroup)
			clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).WithCascadeDeleteOptions(entry.cascadeDeleteOpts).BuildWith(entry.vmExists, true, true, false, nil))

			// create fake factory
			fakeFactory := createFakeFactoryForMachineDeleteWithAPIBehaviorSpecs(g, providerSpec.ResourceGroup, clusterState, entry.rgAccessAPIBehaviorSpec, entry.vmAccessAPIBehaviorSpec, entry.diskAccessAPIBehaviorSpec, entry.nicAccessAPIBehaviorSpec)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := testhelp.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: testhelp.NewMachineObjectMeta(testShootNs, vmName),
			}
			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       testhelp.CreateProviderSecret(),
			})
			if entry.checkErrorFn != nil {
				entry.checkErrorFn(g, err)
			}
			if entry.checkClusterStateFn != nil {
				entry.checkClusterStateFn(g, ctx, clusterState, vmName)
			}
		})
	}
}

func TestDeleteMachineWhenProviderIsNotAzure(t *testing.T) {
	const vmName = "test-vm-0"
	g := NewWithT(t)
	ctx := context.Background()
	fakeFactory := fakes.NewFactory(testResourceGroupName)
	testDriver := NewDefaultDriver(fakeFactory)
	providerSpec := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().Build()
	machineClass, err := testhelp.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
	g.Expect(err).To(BeNil())
	machineClass.Provider = "aws" //set an incorrect provider
	machine := &v1alpha1.Machine{
		ObjectMeta: testhelp.NewMachineObjectMeta(testShootNs, vmName),
	}
	_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
		Machine:      machine,
		MachineClass: machineClass,
		Secret:       testhelp.CreateProviderSecret(),
	})
	g.Expect(err).ToNot(BeNil())
	var statusErr *status.Status
	g.Expect(errors.As(err, &statusErr)).Should(BeTrue())
	g.Expect(statusErr.Code()).To(Equal(codes.InvalidArgument))
}

func TestGetMachineStatus(t *testing.T) {
	table := []struct {
		description            string
		existingVMNames        []string
		targetVMName           string
		shouldOperationSucceed bool
		checkErrorFn           func(g *WithT, err error)
	}{
		{
			"should return an error for a non-existing VM", []string{"vm-0", "vm-1"}, "vm-2", false,
			func(g *WithT, err error) {
				var statusErr *status.Status
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.As(err, &statusErr)).Should(BeTrue())
				g.Expect(statusErr.Code()).To(Equal(codes.NotFound))
			},
		},
		{"should return a valid response for an existing VM", []string{"vm-0", "vm-1"}, "vm-0", true, nil},
	}

	g := NewWithT(t)
	ctx := context.Background()

	// create provider spec
	providerSpec := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().Build()

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// initialize cluster state
			//----------------------------------------------------------------------------
			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec.ResourceGroup)
			for _, vmName := range entry.existingVMNames {
				clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).BuildAllResources())
			}
			// create fake factory
			fakeFactory := fakes.NewFactory(testResourceGroupName)
			vmAccess, err := fakeFactory.NewVirtualMachineAccessBuilder().WithClusterState(clusterState).Build()
			g.Expect(err).To(BeNil())
			fakeFactory.WithVirtualMachineAccess(vmAccess)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := testhelp.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: testhelp.NewMachineObjectMeta(testShootNs, entry.targetVMName),
			}

			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			getMachineStatusResp, err := testDriver.GetMachineStatus(ctx, &driver.GetMachineStatusRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       testhelp.CreateProviderSecret(),
			})
			g.Expect(err == nil).To(Equal(entry.shouldOperationSucceed))
			if err == nil {
				g.Expect(getMachineStatusResp).ToNot(BeNil())
				g.Expect(getMachineStatusResp.NodeName).To(Equal(entry.targetVMName))
				instanceID := helpers.DeriveInstanceID(providerSpec.Location, entry.targetVMName)
				g.Expect(getMachineStatusResp.ProviderID).To(Equal(instanceID))
			}
			if entry.checkErrorFn != nil {
				entry.checkErrorFn(g, err)
			}
		})
	}
}

// unit test helper functions
//------------------------------------------------------------------------------------------------------

func checkError(g *WithT, err error) {
	var statusErr *status.Status
	g.Expect(errors.As(err, &statusErr)).To(BeTrue())
	g.Expect(statusErr.Code()).To(Equal(codes.Internal))
	// TODO: Add additional check when we improve status.Status error type to include the underline error as well.
}

func checkClusterStateAndGetMachineResources(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, expectVMExists bool, expectNICExists bool, expectOSDiskExists bool) fakes.MachineResources {
	vm := checkAndGetVM(g, ctx, factory, vmName, expectVMExists)
	nic := checkAndGetNIC(g, ctx, factory, vmName, expectNICExists)
	osDisk := checkAndGetOSDisk(g, ctx, factory, vmName, expectOSDiskExists)
	return fakes.MachineResources{
		Name:   vmName,
		VM:     vm,
		OSDisk: osDisk,
		NIC:    nic,
	}
}

func createFactoryAndCheckClusterState(g *WithT, ctx context.Context, resourceGroupName string, clusterState *fakes.ClusterState, vmName string, expectVMExists bool, expectNICExists bool, expectOSDiskExists bool) {
	factory := createDefaultFakeFactoryForMachineDelete(g, resourceGroupName, clusterState)
	checkClusterStateAndGetMachineResources(g, ctx, *factory, vmName, expectVMExists, expectNICExists, expectOSDiskExists)
}

func checkCascadeDeleteOptions(t *testing.T, vm armcompute.VirtualMachine, expectedCascadeDeleteOpts fakes.CascadeDeleteOpts) {
	g := NewWithT(t)
	if expectedCascadeDeleteOpts.NIC != nil {
		actualNICDeleteOpt := testhelp.GetCascadeDeleteOptForNIC(vm)
		g.Expect(actualNICDeleteOpt).ToNot(BeNil())
		g.Expect(*actualNICDeleteOpt).To(Equal(*expectedCascadeDeleteOpts.NIC))
	}
	if expectedCascadeDeleteOpts.OSDisk != nil {
		actualOsDiskDeleteOpt := testhelp.GetCascadeDeleteOptForOsDisk(vm)
		g.Expect(actualOsDiskDeleteOpt).ToNot(BeNil())
		g.Expect(*actualOsDiskDeleteOpt).To(Equal(*expectedCascadeDeleteOpts.OSDisk))
	}
	if expectedCascadeDeleteOpts.DataDisk != nil {
		deleteOpts := testhelp.GetCascadeDeleteOptForDataDisks(vm)
		for dataDiskName, actualDeleteOpt := range deleteOpts {
			t.Logf("comparing disk delete option for data disk %s", dataDiskName)
			g.Expect(*actualDeleteOpt).To(Equal(*expectedCascadeDeleteOpts.DataDisk))
		}
	}
}

func checkAndGetVM(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, expectVMExists bool) *armcompute.VirtualMachine {
	vmResp, err := factory.VMAccess.Get(ctx, testResourceGroupName, vmName, nil)
	if expectVMExists {
		g.Expect(err).To(BeNil())
		g.Expect(*vmResp.VirtualMachine.Name).To(Equal(vmName))
		return &vmResp.VirtualMachine
	} else {
		g.Expect(err).ToNot(BeNil())
		g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
		return nil
	}
}

func checkAndGetNIC(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, expectNICExists bool) *armnetwork.Interface {
	nicResp, err := factory.InterfaceAccess.Get(ctx, testResourceGroupName, utils.CreateNICName(vmName), nil)
	if expectNICExists {
		g.Expect(err).To(BeNil())
		g.Expect(nicResp.Interface.Properties.VirtualMachine).ToNot(BeNil())
		return &nicResp.Interface
	} else {
		g.Expect(err).ToNot(BeNil())
		g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
		return nil
	}
}

func checkAndGetOSDisk(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, expectOSDiskExists bool) *armcompute.Disk {
	osDiskResp, err := factory.DisksAccess.Get(ctx, testResourceGroupName, utils.CreateOSDiskName(vmName), nil)
	if expectOSDiskExists {
		g.Expect(err).To(BeNil())
		g.Expect(osDiskResp.ManagedBy).ToNot(BeNil())
		return &osDiskResp.Disk
	} else {
		g.Expect(err).ToNot(BeNil())
		g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
		return nil
	}
}

func createDefaultFakeFactoryForMachineDelete(g *WithT, resourceGroup string, clusterState *fakes.ClusterState) *fakes.Factory {
	return createFakeFactoryForMachineDeleteWithAPIBehaviorSpecs(g, resourceGroup, clusterState, nil, nil, nil, nil)
}

func createFakeFactoryForMachineDeleteWithAPIBehaviorSpecs(g *WithT, resourceGroup string, clusterState *fakes.ClusterState,
	rgAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	vmAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	diskAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	nicAccessAPIBehaviorSpec *fakes.APIBehaviorSpec) *fakes.Factory {

	fakeFactory := fakes.NewFactory(resourceGroup)
	vmAccess, err := fakeFactory.NewVirtualMachineAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(vmAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	nicAccess, err := fakeFactory.NewNICAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(nicAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	rgAccess, err := fakeFactory.NewResourceGroupsAccessBuilder().WithAPIBehaviorSpec(rgAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	diskAccess, err := fakeFactory.NewDiskAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(diskAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	fakeFactory.
		WithVirtualMachineAccess(vmAccess).
		WithResourceGroupsAccess(rgAccess).
		WithNetworkInterfacesAccess(nicAccess).
		WithDisksAccess(diskAccess)

	return fakeFactory
}
