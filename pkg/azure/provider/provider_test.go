// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
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
	corev1 "k8s.io/api/core/v1"
)

const (
	testResourceGroupName = "test-rg"
	testShootNs           = "test-shoot-ns"
	testWorkerPool0Name   = "test-worker-pool-0"
	testDataDiskName      = "test-dd"
)

func TestDeleteMachineWhenVMExists(t *testing.T) {
	table := []struct {
		description                string
		resourceGroup              string                  // used to build initial ClusterState
		existingVMNames            []string                // used to build initial ClusterState
		numDataDisks               int                     // used to build initial ClusterState
		cascadeDeleteOpts          fakes.CascadeDeleteOpts // used to build initial ClusterState
		machineClassResourceGroup  *string                 // for tests where a different resource Group than the one used to create ClusterState needs to be passed.
		targetVMNameToDelete       string                  // name of the VM that will be deleted via DeleteMachine
		shouldDeleteMachineSucceed bool
		checkClusterStateFn        func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string)
	}{
		{
			"should delete all resources(NIC, OSDisk and DataDisks) when cascade delete is set for NIC and all Disks",
			testResourceGroupName,
			[]string{"vm-0", "vm-1"},
			1,
			fakes.CascadeDeleteAllResources,
			nil,
			"vm-1",
			true,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false, dataDiskNames, false, true)
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
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false, dataDiskNames, false, true)
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
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false, dataDiskNames, false, true)
			},
		},
		{
			"should skip delete if the resource group is not found",
			testResourceGroupName,
			[]string{"vm-0", "vm-1"},
			1,
			fakes.CascadeDeleteOpts{},
			to.Ptr("wrong-resource-group"),
			"vm-1",
			true,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, true, true, true, dataDiskNames, true, true)
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
			clusterState := fakes.NewClusterState(providerSpec)
			for _, vmName := range entry.existingVMNames {
				clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).WithCascadeDeleteOptions(entry.cascadeDeleteOpts).BuildAllResources())
			}
			// create fake factory
			fakeFactory := createDefaultFakeFactoryForDeleteMachine(g, providerSpec.ResourceGroup, clusterState)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, entry.machineClassResourceGroup)
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, entry.targetVMNameToDelete),
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
				Secret:       fakes.CreateProviderSecret(),
			})
			g.Expect(err == nil).To(Equal(entry.shouldDeleteMachineSucceed))

			var dataDiskNames []string
			if !utils.IsSliceNilOrEmpty(providerSpec.Properties.StorageProfile.DataDisks) {
				dataDiskNames = make([]string, 0, len(providerSpec.Properties.StorageProfile.DataDisks))
				dataDiskNames = testhelp.CreateDataDiskNames(entry.targetVMNameToDelete, providerSpec)
			}
			// evaluate cluster state post delete machine operation
			entry.checkClusterStateFn(g, ctx, *fakeFactory, entry.targetVMNameToDelete, dataDiskNames)
		})
	}
}

func TestDeleteMachineWhenVMDoesNotExist(t *testing.T) {
	const vmName = "test-vm-0"
	testVMID := fakes.CreateVirtualMachineID(testhelp.SubscriptionID, testResourceGroupName, vmName)

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
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false, dataDiskNames, false, false)
			},
		},
		{
			"should fail delete of NIC when its still associated with a VM",
			true, false, 0, &testVMID, false,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				machineResources := checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, true, false, dataDiskNames, false, true)
				g.Expect(*machineResources.NIC.Properties.VirtualMachine.ID).To(Equal(testVMID))
			},
		},
		{
			"should fail delete of disks when its still associated with a VM",
			false, true, 1, &testVMID, false,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				machineResources := checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, true, dataDiskNames, true, true)
				g.Expect(*machineResources.OSDisk.ManagedBy).To(Equal(testVMID))
				for _, dataDisk := range machineResources.DataDisks {
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
			clusterState := fakes.NewClusterState(providerSpec)
			clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).BuildWith(false, entry.nicPresent, entry.osDiskPresent, entry.numDataDisks > 0, entry.vmID))

			// create fake factory
			fakeFactory := createDefaultFakeFactoryForDeleteMachine(g, providerSpec.ResourceGroup, clusterState)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, vmName),
			}

			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			g.Expect(err == nil).To(Equal(entry.shouldDeleteMachineSucceed))

			dataDiskNames := testhelp.CreateDataDiskNames(vmName, providerSpec)
			entry.checkClusterStateFn(g, ctx, *fakeFactory, vmName, dataDiskNames)
		})
	}
}

func TestDeleteVMInTerminalState(t *testing.T) {
	const vmName = "test-vm-0"

	table := []struct {
		description         string
		cascadeDeleteOpts   fakes.CascadeDeleteOpts
		numDataDisks        int
		checkClusterStateFn func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string)
	}{
		{
			"should delete the VM even if no cascade delete is set for NIC and Disks",
			fakes.CascadeDeleteOpts{}, 1,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false, dataDiskNames, false, false)
			},
		},
		{
			"should delete the VM and all its associated resources when cascade delete is set",
			fakes.CascadeDeleteAllResources, 1,
			func(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, dataDiskNames []string) {
				checkClusterStateAndGetMachineResources(g, ctx, factory, vmName, false, false, false, dataDiskNames, false, false)
			},
		},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			ctx := context.Background()
			// create provider spec
			providerSpecBuilder := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues()
			if entry.numDataDisks > 0 {
				//Add data disks
				providerSpecBuilder.WithDataDisks(testDataDiskName, entry.numDataDisks)
			}
			providerSpec := providerSpecBuilder.Build()
			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec)
			clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).WithCascadeDeleteOptions(entry.cascadeDeleteOpts).BuildWith(true, true, true, true, nil))
			clusterState.MarkVirtualMachineInTerminalState(vmName)

			// create fake factory
			fakeFactory := createDefaultFakeFactoryForDeleteMachine(g, providerSpec.ResourceGroup, clusterState)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, vmName),
			}

			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			g.Expect(err).To(BeNil())
			dataDiskNames := testhelp.CreateDataDiskNames(vmName, providerSpec)
			entry.checkClusterStateFn(g, ctx, *fakeFactory, vmName, dataDiskNames)
		})
	}
}

func TestDeleteExistingVMWithDataDisksInDetachment(t *testing.T) {
	const vmName = "test-vm-0"
	g := NewWithT(t)
	ctx := context.Background()
	// create provider spec
	providerSpec := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().WithDataDisks(testDataDiskName, 2).Build()
	// create cluster state
	clusterState := fakes.NewClusterState(providerSpec)
	clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).WithCascadeDeleteOptions(fakes.CascadeDeleteOpts{}).BuildWith(true, true, true, true, nil))
	g.Expect(clusterState.MarkAllDataDisksInDetachment(vmName)).To(BeTrue())

	// create fake factory
	fakeFactory := createDefaultFakeFactoryForDeleteMachine(g, providerSpec.ResourceGroup, clusterState)

	// Create machine and machine class to be used to create DeleteMachineRequest
	machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
	g.Expect(err).To(BeNil())
	machine := &v1alpha1.Machine{
		ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, vmName),
	}
	// Test
	//----------------------------------------------------------------------------
	testDriver := NewDefaultDriver(fakeFactory)
	_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
		Machine:      machine,
		MachineClass: machineClass,
		Secret:       fakes.CreateProviderSecret(),
	})
	g.Expect(err).To(BeNil())
	g.Expect(clusterState.GetVM(vmName)).To(BeNil())
	_, ok := clusterState.MachineResourcesMap[vmName]
	g.Expect(ok).ToNot(BeTrue())
}

func TestDeleteMachineWithInducedErrors(t *testing.T) {
	const (
		testErrorCode = "test-error-code"
		vmName        = "test-vm-0"
	)

	testInternalServerError := testhelp.InternalServerError(testErrorCode)

	table := []struct {
		description               string
		vmAccessAPIBehaviorSpec   *fakes.APIBehaviorSpec
		rgAccessAPIBehaviorSpec   *fakes.APIBehaviorSpec
		diskAccessAPIBehaviorSpec *fakes.APIBehaviorSpec
		nicAccessAPIBehaviorSpec  *fakes.APIBehaviorSpec
		underlineCause            error
		cascadeDeleteOpts         fakes.CascadeDeleteOpts
		vmExists                  bool
		checkErrorFn              func(g *WithT, err error, underlineCause error)
		checkClusterStateFn       func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string)
	}{
		{
			"should fail when checking resource groups existence returns an error", nil,
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(testResourceGroupName, testhelp.AccessMethodCheckExistence, testInternalServerError),
			nil, nil, testInternalServerError, fakes.CascadeDeleteAllResources, true, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				createFakeFactoryForDeleteMachineAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, true, true, true)
			},
		},
		{
			"should fail when VM access Get call returns an error",
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(vmName, testhelp.AccessMethodGet, testInternalServerError),
			nil, nil, nil, testInternalServerError, fakes.CascadeDeleteAllResources, true, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				createFakeFactoryForDeleteMachineAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, true, true, true)
			},
		},
		{
			"non-existing-vm: should delete left over OSDisk even if error is returned when deleting left over NIC",
			nil, nil, nil,
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(utils.CreateNICName(vmName), testhelp.AccessMethodBeginDelete, testInternalServerError),
			testInternalServerError,
			fakes.CascadeDeleteAllResources, false, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				createFakeFactoryForDeleteMachineAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, false, true, false)
			},
		},
		{
			"non-existing-vm: should delete left over NIC even if there is a panic when deleting left over OSDisk",
			nil, nil,
			fakes.NewAPIBehaviorSpec().AddPanicResourceReaction(utils.CreateOSDiskName(vmName), testhelp.AccessMethodBeginDelete),
			nil, utils.ErrorEncapsulatingPanic, fakes.CascadeDeleteAllResources, false, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				createFakeFactoryForDeleteMachineAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, false, false, true)
			},
		},
		{
			"should fail when existing VM's cascade delete options update returns an error",
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(vmName, testhelp.AccessMethodBeginUpdate, testInternalServerError),
			nil, nil, nil, testInternalServerError, fakes.CascadeDeleteOpts{}, true, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				machineResources := createFakeFactoryForDeleteMachineAndCheckClusterState(g, ctx, testResourceGroupName, clusterState, vmName, true, true, true)
				g.Expect(machineResources.VM).ToNot(BeNil())
				checkCascadeDeleteOptions(t, *machineResources.VM, fakes.CascadeDeleteOpts{})
			},
		},
		{
			"should fail when deletion of the VM post update of cascade deletion option completely fails",
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(vmName, testhelp.AccessMethodBeginDelete, testInternalServerError),
			nil, nil, nil, testInternalServerError, fakes.CascadeDeleteOpts{}, true, checkError,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				factory := createDefaultFakeFactoryForDeleteMachine(g, testResourceGroupName, clusterState)
				machineResources := checkClusterStateAndGetMachineResources(g, ctx, *factory, vmName, true, true, true, nil, false, true)
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
			clusterState := fakes.NewClusterState(providerSpec)
			clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).WithCascadeDeleteOptions(entry.cascadeDeleteOpts).BuildWith(entry.vmExists, true, true, false, nil))

			// create fake factory
			fakeFactory := createFakeFactoryForDeleteMachineWithAPIBehaviorSpecs(g, providerSpec.ResourceGroup, clusterState, entry.rgAccessAPIBehaviorSpec, entry.vmAccessAPIBehaviorSpec, entry.diskAccessAPIBehaviorSpec, entry.nicAccessAPIBehaviorSpec)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, vmName),
			}
			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			if entry.checkErrorFn != nil {
				entry.checkErrorFn(g, err, entry.underlineCause)
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
	machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
	g.Expect(err).To(BeNil())
	machineClass.Provider = "aws" //set an incorrect provider
	machine := &v1alpha1.Machine{
		ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, vmName),
	}
	_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
		Machine:      machine,
		MachineClass: machineClass,
		Secret:       fakes.CreateProviderSecret(),
	})
	g.Expect(err).ToNot(BeNil())
	var statusErr *status.Status
	g.Expect(errors.As(err, &statusErr)).To(BeTrue())
	g.Expect(statusErr.Code()).To(Equal(codes.InvalidArgument))
}

func TestGetMachineStatus(t *testing.T) {
	const testErrorCode = "test-error-code"
	testInternalServerError := testhelp.InternalServerError(testErrorCode)

	table := []struct {
		description            string
		existingVMNames        []string
		targetVMName           string
		shouldOperationSucceed bool
		vmAccessApiBehavior    *fakes.APIBehaviorSpec
		checkErrorFn           func(g *WithT, err error)
	}{
		{
			"should fail when VM access Get call returns an error",
			[]string{"vm-0", "vm-1"},
			"vm-1",
			false,
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction("vm-1", testhelp.AccessMethodGet, testInternalServerError),
			func(g *WithT, err error) {
				checkError(g, err, testInternalServerError)
			},
		},
		{
			"should return an error for a non-existing VM", []string{"vm-0", "vm-1"}, "vm-2", false, nil,
			func(g *WithT, err error) {
				var statusErr *status.Status
				g.Expect(err).ToNot(BeNil())
				g.Expect(errors.As(err, &statusErr)).Should(BeTrue())
				g.Expect(statusErr.Code()).To(Equal(codes.NotFound))
			},
		},
		{"should return a valid response for an existing VM", []string{"vm-0", "vm-1"}, "vm-0", true, nil, nil},
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
			clusterState := fakes.NewClusterState(providerSpec)
			for _, vmName := range entry.existingVMNames {
				clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).BuildAllResources())
			}
			// create fake factory and initialize vmAccess only
			fakeFactory := fakes.NewFactory(testResourceGroupName)
			vmAccess, err := fakeFactory.NewVirtualMachineAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(entry.vmAccessApiBehavior).Build()
			g.Expect(err).To(BeNil())
			fakeFactory.WithVirtualMachineAccess(vmAccess)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, entry.targetVMName),
			}

			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			getMachineStatusResp, err := testDriver.GetMachineStatus(ctx, &driver.GetMachineStatusRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			g.Expect(err == nil).To(Equal(entry.shouldOperationSucceed))
			if err == nil {
				g.Expect(getMachineStatusResp).ToNot(BeNil())
				g.Expect(getMachineStatusResp.NodeName).To(Equal(entry.targetVMName))
				expectedInstanceID := helpers.DeriveInstanceID(providerSpec.Location, entry.targetVMName)
				g.Expect(getMachineStatusResp.ProviderID).To(Equal(expectedInstanceID))
			}
			if entry.checkErrorFn != nil {
				entry.checkErrorFn(g, err)
			}
		})
	}
}

func TestListMachines(t *testing.T) {
	type machineResourcesTestSpec struct {
		vmName          string
		vmPresent       bool
		osDiskPresent   bool
		dataDiskPresent bool
		nicPresent      bool
		// if present this will overwrite the tags that are borrowed from provider spec for this machine resource VM. Only specify this when vmPresent is true else it will never reflect.
		vmTags map[string]string
		// if present this will overwrite the tags that are borrowed from provider spec for this machine resource NIC. Only specify this when nicPresent is true else it will never reflect.
		nicTags map[string]string
		// if present this will overwrite the tags that are borrowed from provider spec for this machine resource OSDisk. Only specify this when nicPresent is true else it will never reflect.
		osDiskTags map[string]string
		// if present this will overwrite the tags that are borrowed from provider spec for this machine resource DataDisks. Only specify this when nicPresent is true else it will never reflect.
		dataDiskTags map[string]string
	}

	const nonMatchingShootNs = "non-matching-shoot-ns"

	nonMatchingTags := map[string]string{
		"kubernetes.io-cluster-" + nonMatchingShootNs: "1",
		"kubernetes.io-role-node":                     "1",
	}

	table := []struct {
		description     string
		mrTestSpecs     []machineResourcesTestSpec
		apiBehaviorSpec *fakes.APIBehaviorSpec
		numDataDisks    int
		expectedResult  []string
		expectedErr     bool
	}{
		{
			"should return no result if no resources exist", nil, nil, 0, []string{}, false,
		},
		{
			"should return all vm names where vms exist",
			[]machineResourcesTestSpec{
				{"vm-0", true, true, true, true, nil, nil, nil, nil},
				{"vm-1", true, true, true, true, nil, nil, nil, nil},
			}, nil, 1, []string{"vm-0", "vm-1"}, false,
		},
		{
			"should return vm names only for vms where vm does not exist but a nic exists",
			[]machineResourcesTestSpec{
				{"vm-0", false, false, false, false, nil, nil, nil, nil},
				{"vm-1", false, false, true, false, nil, nil, nil, nil},
			}, nil, 1, []string{"vm-1"}, false,
		},
		{
			"should return vm names only for vms where vm does not exist but an OSDisk exists",
			[]machineResourcesTestSpec{
				{"vm-0", false, false, false, false, nil, nil, nil, nil},
				{"vm-1", false, true, false, false, nil, nil, nil, nil},
			}, nil, 2, []string{"vm-1"}, false,
		},
		{
			"should only return vms matching mandatory provider spec tags",
			[]machineResourcesTestSpec{
				{"vm-0", true, true, true, true, nonMatchingTags, nonMatchingTags, nonMatchingTags, nonMatchingTags},
				{"vm-1", true, true, true, true, nil, nil, nil, nil},
				{"vm-2", true, true, true, false, nonMatchingTags, nonMatchingTags, nonMatchingTags, nonMatchingTags},
				{"vm-3", true, true, true, false, nil, nil, nil, nil},
				{"vm-4", true, true, true, true, nonMatchingTags, nil, nonMatchingTags, nil},
			}, nil, 1, []string{"vm-1", "vm-3", "vm-4"}, false,
		},
	}

	g := NewWithT(t)
	ctx := context.Background()

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// create provider spec
			providerSpecBuilder := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues()
			if entry.numDataDisks > 0 {
				//Add data disks
				providerSpecBuilder.WithDataDisks(testDataDiskName, entry.numDataDisks)
			}
			providerSpec := providerSpecBuilder.Build()
			// initialize cluster state
			//----------------------------------------------------------------------------
			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec)
			if entry.mrTestSpecs != nil {
				for _, mrTestSpec := range entry.mrTestSpecs {
					var testVMID *string
					if !mrTestSpec.vmPresent {
						testVMID = to.Ptr(fakes.CreateVirtualMachineID(testhelp.SubscriptionID, testResourceGroupName, mrTestSpec.vmName))
					}
					mr := fakes.NewMachineResourcesBuilder(providerSpec, mrTestSpec.vmName).BuildWith(mrTestSpec.vmPresent, mrTestSpec.nicPresent, mrTestSpec.osDiskPresent, mrTestSpec.dataDiskPresent, testVMID)
					if mrTestSpec.vmPresent && mrTestSpec.vmTags != nil {
						mr.VM.Tags = utils.CreateResourceTags(mrTestSpec.vmTags)
					}
					if mrTestSpec.nicPresent && mrTestSpec.nicTags != nil {
						mr.NIC.Tags = utils.CreateResourceTags(mrTestSpec.nicTags)
					}
					if mrTestSpec.osDiskPresent && mrTestSpec.osDiskTags != nil {
						mr.OSDisk.Tags = utils.CreateResourceTags(mrTestSpec.osDiskTags)
					}
					if mrTestSpec.dataDiskPresent && mrTestSpec.dataDiskTags != nil {
						for _, dataDisk := range mr.DataDisks {
							dataDisk.Tags = utils.CreateResourceTags(mrTestSpec.dataDiskTags)
						}
					}
					clusterState.AddMachineResources(mr)
				}
			}

			// create fake factory
			fakeFactory := createDefaultFakeFactoryForListMachines(g, testResourceGroupName, clusterState, nil)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())

			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			listMachinesResp, err := testDriver.ListMachines(ctx, &driver.ListMachinesRequest{
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			g.Expect(err != nil).To(Equal(entry.expectedErr))
			actualVMNames := getVMNamesFromListMachineResponse(listMachinesResp)
			g.Expect(fakes.ActualSliceEqualsExpectedSlice(actualVMNames, entry.expectedResult)).To(BeTrue())
		})
	}
}

func TestListMachineWithInducedErrors(t *testing.T) {
	const (
		vmName        = "test-vm-0"
		testErrorCode = "test-error-code"
	)
	testInternalServerError := testhelp.InternalServerError(testErrorCode)

	table := []struct {
		description     string
		apiBehaviorSpec *fakes.APIBehaviorSpec
	}{
		{
			"should fail listing machines when resource-graph query for VM resource type returns error",
			fakes.NewAPIBehaviorSpec().AddErrorResourceTypeReaction(utils.VirtualMachinesResourceType, testhelp.AccessMethodResources, testInternalServerError),
		},
		{
			"should fail listing machines when resource-graph query for NIC resource type returns error",
			fakes.NewAPIBehaviorSpec().AddErrorResourceTypeReaction(utils.NetworkInterfacesResourceType, testhelp.AccessMethodResources, testInternalServerError),
		},
	}

	g := NewWithT(t)
	ctx := context.Background()

	// create provider spec
	providerSpec := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().Build()
	clusterState := fakes.NewClusterState(providerSpec)
	clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).BuildWith(true, true, true, false, nil))

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// create fake factory
			fakeFactory := createDefaultFakeFactoryForListMachines(g,
				testResourceGroupName,
				clusterState,
				entry.apiBehaviorSpec,
			)
			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())

			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			_, err = testDriver.ListMachines(ctx, &driver.ListMachinesRequest{
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			g.Expect(err).ToNot(BeNil())
			checkError(g, err, testInternalServerError)
		})
	}
}

func TestGetVolumeIDs(t *testing.T) {
	table := []struct {
		description                     string
		existingAzureDiskVolSourceNames []string
		existingAzureCSIVolHandles      []string
		existingNonAzureCSIVolHandles   []string
		expectedVolumeIDs               []string
	}{
		{"should return empty volumeIDs when no pv exist", nil, nil, nil, []string{}},
		{"should return empty volumeIDS when only non-csi vol sources are defined", nil, nil, []string{"non-az-csi-vol-1", "non-az-csi-vol-2"}, []string{}},
		{"should return azure disk vol sources when defined", []string{"az-disk-1", "az-disk-2"}, nil, []string{"non-az-csi-vol-1"}, []string{"az-disk-1", "az-disk-2"}},
		{"should return azure csi vol sources when defined", nil, []string{"az-csi-vol-1", "az-csi-vol-2"}, []string{"non-az-csi-vol-1"}, []string{"az-csi-vol-1", "az-csi-vol-2"}},
		{"should return azure disk and csi vol sources when defined", []string{"az-disk-1", "az-disk-2"}, []string{"az-csi-vol-1", "az-csi-vol-2"}, []string{"non-az-csi-vol-1"}, []string{"az-disk-1", "az-disk-2", "az-csi-vol-1", "az-csi-vol-2"}},
	}

	g := NewWithT(t)
	ctx := context.Background()
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			var pvSpecs []*corev1.PersistentVolumeSpec
			for _, diskVolSrcName := range entry.existingAzureDiskVolSourceNames {
				pvSpec := &corev1.PersistentVolumeSpec{
					PersistentVolumeSource: fakes.CreateAzureDiskPVSource(testResourceGroupName, diskVolSrcName),
				}
				pvSpecs = append(pvSpecs, pvSpec)
			}
			for _, azCSIVolHandle := range entry.existingAzureCSIVolHandles {
				pvSpec := &corev1.PersistentVolumeSpec{
					PersistentVolumeSource: fakes.CreateCSIPVSource(utils.AzureCSIDriverName, azCSIVolHandle),
				}
				pvSpecs = append(pvSpecs, pvSpec)
			}
			for _, nonAzCSIVolHandle := range entry.existingNonAzureCSIVolHandles {
				pvSpec := &corev1.PersistentVolumeSpec{
					PersistentVolumeSource: fakes.CreateCSIPVSource("test-non-az-driver", nonAzCSIVolHandle),
				}
				pvSpecs = append(pvSpecs, pvSpec)
			}
			testDriver := NewDefaultDriver(fakes.NewFactory(testResourceGroupName))
			resp, err := testDriver.GetVolumeIDs(ctx, &driver.GetVolumeIDsRequest{PVSpecs: pvSpecs})
			g.Expect(err).To(BeNil())
			g.Expect(fakes.ActualSliceEqualsExpectedSlice(resp.VolumeIDs, entry.expectedVolumeIDs))
		})
	}
}

// TestCreateMachineWhenPrerequisitesFail tests all cases where one or more Azure API calls made to get prerequisite
// resources fail. Prerequisites consist of the following activities:
// 1. Get Subnet
// 2. Get VM Image
// If VM Image is a marketplace image then also do the following:
// 3. Get AgreementTerms
// 4. If not accepted then accept and update AgreementTerms
// If any of the above-mentioned prerequisites fail, then it should not result in creation of any resources for the machine.
func TestCreateMachineWhenPrerequisitesFail(t *testing.T) {
	const (
		vmName                = "vm-0"
		internalServerErrCode = "test-error-code"
	)
	subnetName := fakes.CreateSubnetName(testShootNs)
	vnetName := testShootNs
	internalServerErr := testhelp.InternalServerError(internalServerErrCode)
	table := []struct {
		description                        string
		subnetName                         string
		vnetName                           string
		subnetResourceGroup                *string // If specified then this resource-group will be used to create a subnet resource in ClusterState, else resource group at the providerSpec will be used.
		providerSpecVnetResourceGroup      *string // If specified this will be used to set the vnet resource group in provider spec, else resource group at the providerSpec will be used.
		subnetExists                       bool
		vmImageExists                      bool
		agreementExists                    bool
		agreementAccepted                  bool
		vmAccessAPIBehavior                *fakes.APIBehaviorSpec
		subnetAccessAPIBehavior            *fakes.APIBehaviorSpec
		vmImageAccessAPIBehavior           *fakes.APIBehaviorSpec
		mktPlaceAgreementAccessAPIBehavior *fakes.APIBehaviorSpec
		checkErrorFn                       func(g *WithT, clusterState *fakes.ClusterState, err error)
	}{
		{
			"should fail machine creation when no subnet with given name exists, no resources should be created", subnetName, vnetName, nil, nil,
			false, true, true, true,
			nil, nil, nil, nil,
			func(g *WithT, clusterState *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.NotFound)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusNotFound))
				g.Expect(azRespErr.ErrorCode).To(Equal(testhelp.ErrorCodeSubnetNotFound))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodGet))
				g.Expect(fakes.IsSubnetURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, fakes.SubnetSpec{
					ResourceGroup: clusterState.ProviderSpec.ResourceGroup,
					SubnetName:    subnetName,
					VnetName:      vnetName,
				})).To(BeTrue())
			},
		},
		{
			"should fail machine creation when subnet GET fails, no resources should be created", subnetName, vnetName, nil, nil,
			true, true, true, true,
			nil, fakes.NewAPIBehaviorSpec().AddErrorResourceTypeReaction(utils.SubnetResourceType, testhelp.AccessMethodGet, internalServerErr), nil, nil,
			func(g *WithT, clusterState *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.Internal)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusInternalServerError))
				g.Expect(azRespErr.ErrorCode).To(Equal(internalServerErrCode))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodGet))
				g.Expect(fakes.IsSubnetURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, *clusterState.SubnetSpec)).To(BeTrue())
			},
		},
		{
			"should fail machine creation when resource group for subnet does not exist, no resources should be created", subnetName, vnetName, to.Ptr("vnet-rg"), to.Ptr("provider-spec-vnet-rg"),
			true, true, true, true,
			nil, nil, nil, nil,
			func(g *WithT, clusterState *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.NotFound)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusNotFound))
				g.Expect(azRespErr.ErrorCode).To(Equal(testhelp.ErrorCodeResourceGroupNotFound))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodGet))
				subnetSpec := *clusterState.SubnetSpec
				subnetSpec.ResourceGroup = "provider-spec-vnet-rg"
				g.Expect(fakes.IsSubnetURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, subnetSpec)).To(BeTrue())
			},
		},
		{
			"should fail machine creation when VM Image is not found, no resources should be created", subnetName, vnetName, nil, nil,
			true, false, false, false,
			nil, nil, nil, nil,
			func(g *WithT, clusterState *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.NotFound)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusNotFound))
				g.Expect(azRespErr.ErrorCode).To(Equal(testhelp.ErrorCodeVMImageNotFound))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodGet))
				publisher, offer, sku, version := fakes.GetDefaultVMImageParts()
				g.Expect(fakes.IsVMImageURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, clusterState.ProviderSpec.Location, fakes.VMImageSpec{
					Publisher: publisher,
					Offer:     offer,
					SKU:       sku,
					Version:   version,
				})).To(BeTrue())
			},
		},
		{
			"should fail machine creation when VM Image GET fails, no resources should be created", subnetName, vnetName, nil, nil,
			true, true, true, true,
			nil, nil, fakes.NewAPIBehaviorSpec().AddErrorResourceTypeReaction(utils.VMImageResourceType, testhelp.AccessMethodGet, internalServerErr), nil,
			func(g *WithT, clusterState *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.Internal)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusInternalServerError))
				g.Expect(azRespErr.ErrorCode).To(Equal(internalServerErrCode))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodGet))
				publisher, offer, sku, version := fakes.GetDefaultVMImageParts()
				g.Expect(fakes.IsVMImageURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, clusterState.ProviderSpec.Location, fakes.VMImageSpec{
					Publisher: publisher,
					Offer:     offer,
					SKU:       sku,
					Version:   version,
				})).To(BeTrue())
			},
		},
		{
			"should fail machine creation when there is no agreement for the VM Image, no resources should be created", subnetName, vnetName, nil, nil,
			true, true, false, false,
			nil, nil, nil, nil,
			func(g *WithT, _ *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.NotFound)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusBadRequest))
				g.Expect(azRespErr.ErrorCode).To(Equal(testhelp.ErrorCodeBadRequest))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodGet))
				publisher, offer, sku, version := fakes.GetDefaultVMImageParts()
				g.Expect(fakes.IsMktPlaceAgreementURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, fakes.VMImageSpec{
					Publisher: publisher,
					Offer:     offer,
					SKU:       sku,
					Version:   version,
				})).To(BeTrue())
			},
		},
		{
			"should fail machine creation when update of agreement for VM Image fails, no resources should be created", subnetName, vnetName, nil, nil,
			true, true, true, false,
			nil, nil, nil, fakes.NewAPIBehaviorSpec().AddErrorResourceTypeReaction(utils.MarketPlaceOrderingOfferType, testhelp.AccessMethodCreate, testhelp.InternalServerError("test-error-code")),
			func(g *WithT, _ *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.Internal)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusInternalServerError))
				g.Expect(azRespErr.ErrorCode).To(Equal(internalServerErrCode))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodPut))
				publisher, offer, sku, version := fakes.GetDefaultVMImageParts()
				g.Expect(fakes.IsMktPlaceAgreementURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, fakes.VMImageSpec{
					Publisher: publisher,
					Offer:     offer,
					SKU:       sku,
					Version:   version,
				})).To(BeTrue())
			},
		},
	}

	g := NewWithT(t)
	ctx := context.Background()

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// create provider spec
			providerSpecBuilder := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues()
			if entry.providerSpecVnetResourceGroup != nil {
				providerSpecBuilder.WithSubnetInfo(*entry.providerSpecVnetResourceGroup)
			}
			providerSpec := providerSpecBuilder.Build()

			// initialize cluster state
			//----------------------------------------------------------------------------
			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec)
			if entry.vmImageExists {
				clusterState.WithDefaultVMImageSpec()
			}
			if entry.agreementExists {
				clusterState.WithAgreementTerms(entry.agreementAccepted)
			}
			if entry.subnetExists {
				vnetResourceGroup := providerSpec.ResourceGroup
				if entry.subnetResourceGroup != nil {
					vnetResourceGroup = *entry.subnetResourceGroup
				}
				clusterState.WithSubnet(vnetResourceGroup, entry.subnetName, entry.vnetName)
			}
			// create fake factory
			fakeFactory := createFakeFactoryForCreateMachineWithAPIBehaviorSpecs(g, providerSpec.ResourceGroup, clusterState, entry.vmAccessAPIBehavior, entry.subnetAccessAPIBehavior, nil, entry.vmImageAccessAPIBehavior, entry.mktPlaceAgreementAccessAPIBehavior)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, vmName),
			}
			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			_, err = testDriver.CreateMachine(ctx, &driver.CreateMachineRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			checkClusterStateAndGetMachineResources(g, ctx, *fakeFactory, vmName, false, false, false, nil, false, false)
			if entry.checkErrorFn != nil {
				entry.checkErrorFn(g, clusterState, err)
			}
		})
	}
}

func TestCreateMachineWhenNICOrVMCreationFails(t *testing.T) {
	const (
		vmName                = "vm-0"
		internalServerErrCode = "test-error-code"
	)
	nicName := utils.CreateNICName(vmName)
	internalServerErr := testhelp.InternalServerError(internalServerErrCode)
	ctx := context.Background()

	table := []struct {
		description          string
		nicExists            bool
		vmAccessAPIBehavior  *fakes.APIBehaviorSpec
		nicAccessAPIBehavior *fakes.APIBehaviorSpec
		checkClusterStateFn  func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string)
		checkErrorFn         func(g *WithT, clusterState *fakes.ClusterState, err error)
	}{
		{
			// NIC GET calls does not return error in case it is NotFound
			"should fail machine creation with NIC GET fails, no resources should be created", true,
			nil, fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(nicName, testhelp.AccessMethodGet, internalServerErr),
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				factory := createDefaultFakeFactoryForCreateMachine(g, clusterState)
				checkClusterStateAndGetMachineResources(g, ctx, *factory, vmName, false, true, false, nil, false, false)
			},
			func(g *WithT, clusterState *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.Internal)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusInternalServerError))
				g.Expect(azRespErr.ErrorCode).To(Equal(internalServerErrCode))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodGet))
				g.Expect(fakes.IsNicURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, clusterState.ProviderSpec.ResourceGroup, nicName)).To(BeTrue())
			},
		},
		{
			"should fail machine creation when NIC creation fails, no resources should be created", false,
			nil, fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(nicName, testhelp.AccessMethodBeginCreateOrUpdate, internalServerErr),
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				factory := createDefaultFakeFactoryForCreateMachine(g, clusterState)
				checkClusterStateAndGetMachineResources(g, ctx, *factory, vmName, false, false, false, nil, false, false)
			},
			func(g *WithT, clusterState *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.Internal)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusInternalServerError))
				g.Expect(azRespErr.ErrorCode).To(Equal(internalServerErrCode))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodPut))
				g.Expect(fakes.IsNicURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, clusterState.ProviderSpec.ResourceGroup, nicName)).To(BeTrue())
			},
		},
		{
			"should fail machine creation when VM creation fails, only NIC resource should now exist", false,
			fakes.NewAPIBehaviorSpec().AddErrorResourceReaction(vmName, testhelp.AccessMethodBeginCreateOrUpdate, internalServerErr), nil,
			func(g *WithT, ctx context.Context, clusterState *fakes.ClusterState, vmName string) {
				factory := createDefaultFakeFactoryForCreateMachine(g, clusterState)
				checkClusterStateAndGetMachineResources(g, ctx, *factory, vmName, false, true, false, nil, false, false)
			},
			func(g *WithT, clusterState *fakes.ClusterState, err error) {
				azRespErr := checkAndGetWrapperAzResponseError(g, err, codes.Internal)
				g.Expect(azRespErr.StatusCode).To(Equal(http.StatusInternalServerError))
				g.Expect(azRespErr.ErrorCode).To(Equal(internalServerErrCode))
				g.Expect(azRespErr.RawResponse.Request.Method).To(Equal(http.MethodPut))
				g.Expect(fakes.IsVMURIPath(azRespErr.RawResponse.Request.URL.Path, testhelp.SubscriptionID, clusterState.ProviderSpec.ResourceGroup, vmName)).To(BeTrue())
			},
		},
	}

	g := NewWithT(t)
	providerSpec := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().Build()

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// initialize cluster state
			//----------------------------------------------------------------------------
			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec)
			if entry.nicExists {
				clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).BuildWith(false, true, false, false, nil))
			}
			clusterState.WithDefaultVMImageSpec().WithAgreementTerms(true).WithSubnet(providerSpec.ResourceGroup, fakes.CreateSubnetName(testShootNs), testShootNs)
			// create fake factory
			fakeFactory := createFakeFactoryForCreateMachineWithAPIBehaviorSpecs(g, providerSpec.ResourceGroup, clusterState, entry.vmAccessAPIBehavior, nil, entry.nicAccessAPIBehavior, nil, nil)

			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			g.Expect(err).To(BeNil())
			machine := &v1alpha1.Machine{
				ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, vmName),
			}
			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			_, err = testDriver.CreateMachine(ctx, &driver.CreateMachineRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			if entry.checkClusterStateFn != nil {
				entry.checkClusterStateFn(g, ctx, clusterState, vmName)
			}
			if entry.checkErrorFn != nil {
				entry.checkErrorFn(g, clusterState, err)
			}
		})
	}
}

func TestSuccessfulCreationOfMachine(t *testing.T) {
	providerSpecBuilder := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).
		WithDefaultValues().
		WithDataDisks(testDataDiskName, 2)
	providerSpec := providerSpecBuilder.Build()

	table := []struct {
		description      string
		withPurchasePlan bool
	}{
		{"should create machine successfully if purchase plan is present", true},
		{"should create machine successfully if purchase plan is not present", false},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			// initialize cluster state
			//----------------------------------------------------------------------------
			// create cluster state
			clusterState := fakes.NewClusterState(providerSpec)
			publisher, offer, sku, version := fakes.GetDefaultVMImageParts()
			vmImageSpec := fakes.VMImageSpec{
				Publisher:  publisher,
				Offer:      offer,
				SKU:        sku,
				Version:    version,
				PlanExists: entry.withPurchasePlan,
			}
			clusterState.WithVMImageSpec(vmImageSpec).WithAgreementTerms(true).WithSubnet(providerSpec.ResourceGroup, fakes.CreateSubnetName(testShootNs), testShootNs)
			// create fake factory
			fakeFactory := createFakeFactoryForCreateMachineWithAPIBehaviorSpecs(g, providerSpec.ResourceGroup, clusterState, nil, nil, nil, nil, nil)
			// Create machine and machine class to be used to create DeleteMachineRequest
			machineClass, err := fakes.CreateMachineClass(providerSpec, to.Ptr(testResourceGroupName))
			const vmName = "vm-0"
			g.Expect(err).To(BeNil())
			ctx := context.Background()
			machine := &v1alpha1.Machine{
				ObjectMeta: fakes.NewMachineObjectMeta(testShootNs, vmName),
			}
			dataDiskNames := testhelp.CreateDataDiskNames(vmName, providerSpec)

			// Test
			//----------------------------------------------------------------------------
			testDriver := NewDefaultDriver(fakeFactory)
			resp, err := testDriver.CreateMachine(ctx, &driver.CreateMachineRequest{
				Machine:      machine,
				MachineClass: machineClass,
				Secret:       fakes.CreateProviderSecret(),
			})
			g.Expect(err).To(BeNil())
			checkClusterStateAndGetMachineResources(g, ctx, *fakeFactory, vmName, true, true, true, dataDiskNames, true, true)
			g.Expect(resp.NodeName).To(Equal(vmName))
			expectedProviderID := helpers.DeriveInstanceID(providerSpec.Location, vmName)
			g.Expect(resp.ProviderID).To(Equal(expectedProviderID))
		})
	}
}

// unit test helper functions
//------------------------------------------------------------------------------------------------------

func checkError(g *WithT, err error, underlineCause error) {
	var statusErr *status.Status
	g.Expect(errors.As(err, &statusErr)).To(BeTrue())
	g.Expect(statusErr.Code()).To(Equal(codes.Internal))
	g.Expect(errors.Is(statusErr.Cause(), underlineCause)).To(BeTrue())
}

func checkClusterStateAndGetMachineResources(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, expectVMExists bool, expectNICExists bool, expectOSDiskExists bool, expectedDataDiskNames []string, expectDataDiskExists bool, expectAssociatedVMID bool) fakes.MachineResources {
	vm := checkAndGetVM(g, ctx, factory, vmName, expectVMExists)
	nic := checkAndGetNIC(g, ctx, factory, vmName, expectNICExists, expectAssociatedVMID)
	osDisk := checkAndGetOSDisk(g, ctx, factory, vmName, expectOSDiskExists, expectAssociatedVMID)
	dataDisks := checkAndGetDataDisks(g, ctx, factory, expectedDataDiskNames, expectDataDiskExists, expectAssociatedVMID)
	return fakes.MachineResources{
		Name:      vmName,
		VM:        vm,
		OSDisk:    osDisk,
		DataDisks: dataDisks,
		NIC:       nic,
	}
}

func createFakeFactoryForDeleteMachineAndCheckClusterState(g *WithT, ctx context.Context, resourceGroupName string, clusterState *fakes.ClusterState, vmName string, expectVMExists bool, expectNICExists bool, expectOSDiskExists bool) fakes.MachineResources {
	factory := createDefaultFakeFactoryForDeleteMachine(g, resourceGroupName, clusterState)
	return checkClusterStateAndGetMachineResources(g, ctx, *factory, vmName, expectVMExists, expectNICExists, expectOSDiskExists, []string{}, false, false)
}

func createDefaultFakeFactoryForListMachines(g *WithT, resourceGroup string, clusterState *fakes.ClusterState, resourceGraphAccessBehaviorSpec *fakes.APIBehaviorSpec) *fakes.Factory {
	fakeFactory := fakes.NewFactory(resourceGroup)
	resourceGraphAccess, err := fakeFactory.NewResourceGraphAccessBuilder().
		WithClusterState(clusterState).
		WithAPIBehaviorSpec(resourceGraphAccessBehaviorSpec).
		Build()
	g.Expect(err).To(BeNil())
	fakeFactory.WithResourceGraphAccess(resourceGraphAccess)
	return fakeFactory
}

func checkCascadeDeleteOptions(t *testing.T, vm armcompute.VirtualMachine, expectedCascadeDeleteOpts fakes.CascadeDeleteOpts) {
	g := NewWithT(t)
	if expectedCascadeDeleteOpts.NIC != nil {
		actualNICDeleteOpt := fakes.GetCascadeDeleteOptForNIC(vm)
		g.Expect(actualNICDeleteOpt).ToNot(BeNil())
		g.Expect(*actualNICDeleteOpt).To(Equal(*expectedCascadeDeleteOpts.NIC))
	}
	if expectedCascadeDeleteOpts.OSDisk != nil {
		actualOsDiskDeleteOpt := fakes.GetCascadeDeleteOptForOsDisk(vm)
		g.Expect(actualOsDiskDeleteOpt).ToNot(BeNil())
		g.Expect(*actualOsDiskDeleteOpt).To(Equal(*expectedCascadeDeleteOpts.OSDisk))
	}
	if expectedCascadeDeleteOpts.DataDisk != nil {
		deleteOpts := fakes.GetCascadeDeleteOptForDataDisks(vm)
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

func checkAndGetNIC(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, expectNICExists bool, expectAssociatedVMID bool) *armnetwork.Interface {
	nicResp, err := factory.InterfaceAccess.Get(ctx, testResourceGroupName, utils.CreateNICName(vmName), nil)
	if expectNICExists {
		g.Expect(err).To(BeNil())
		if expectAssociatedVMID {
			g.Expect(nicResp.Interface.Properties.VirtualMachine).ToNot(BeNil())
			g.Expect(nicResp.Interface.Properties.VirtualMachine.ID).ToNot(BeNil())
		}
		return &nicResp.Interface
	} else {
		g.Expect(err).ToNot(BeNil())
		g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
		return nil
	}
}

func checkAndGetOSDisk(g *WithT, ctx context.Context, factory fakes.Factory, vmName string, expectOSDiskExists bool, expectAssociatedVMID bool) *armcompute.Disk {
	osDiskResp, err := factory.DisksAccess.Get(ctx, testResourceGroupName, utils.CreateOSDiskName(vmName), nil)
	if expectOSDiskExists {
		g.Expect(err).To(BeNil())
		if expectAssociatedVMID {
			g.Expect(osDiskResp.ManagedBy).ToNot(BeNil())
		}
		return &osDiskResp.Disk
	} else {
		g.Expect(err).ToNot(BeNil())
		g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
		return nil
	}
}

func checkAndGetDataDisks(g *WithT, ctx context.Context, factory fakes.Factory, expectedDataDiskNames []string, expectDataDisksExists bool, expectedAssociatedVMID bool) map[string]*armcompute.Disk {
	dataDisks := make(map[string]*armcompute.Disk)
	if expectedDataDiskNames == nil {
		return dataDisks
	}
	for _, dataDiskName := range expectedDataDiskNames {
		dataDisk, err := factory.DisksAccess.Get(ctx, testResourceGroupName, dataDiskName, nil)
		if expectDataDisksExists {
			g.Expect(err).To(BeNil())
			if expectedAssociatedVMID {
				g.Expect(dataDisk.ManagedBy).ToNot(BeNil())
			}
			dataDisks[dataDiskName] = &dataDisk.Disk
		} else {
			g.Expect(err).ToNot(BeNil())
			g.Expect(accesserrors.IsNotFoundAzAPIError(err)).To(BeTrue())
		}
	}
	return dataDisks
}

func createDefaultFakeFactoryForDeleteMachine(g *WithT, resourceGroup string, clusterState *fakes.ClusterState) *fakes.Factory {
	return createFakeFactoryForDeleteMachineWithAPIBehaviorSpecs(g, resourceGroup, clusterState, nil, nil, nil, nil)
}

func createFakeFactoryForDeleteMachineWithAPIBehaviorSpecs(g *WithT, resourceGroup string, clusterState *fakes.ClusterState,
	rgAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	vmAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	diskAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	nicAccessAPIBehaviorSpec *fakes.APIBehaviorSpec) *fakes.Factory {

	factory := fakes.NewFactory(resourceGroup)
	vmAccess, err := factory.NewVirtualMachineAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(vmAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	nicAccess, err := factory.NewNICAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(nicAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	rgAccess, err := factory.NewResourceGroupsAccessBuilder().WithAPIBehaviorSpec(rgAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	diskAccess, err := factory.NewDiskAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(diskAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	factory.
		WithVirtualMachineAccess(vmAccess).
		WithResourceGroupsAccess(rgAccess).
		WithNetworkInterfacesAccess(nicAccess).
		WithDisksAccess(diskAccess)

	return factory
}

func createDefaultFakeFactoryForCreateMachine(g *WithT, clusterState *fakes.ClusterState) *fakes.Factory {
	return createFakeFactoryForCreateMachineWithAPIBehaviorSpecs(g, clusterState.ProviderSpec.ResourceGroup, clusterState, nil, nil, nil, nil, nil)
}

func createFakeFactoryForCreateMachineWithAPIBehaviorSpecs(g *WithT, resourceGroup string, clusterState *fakes.ClusterState,
	vmAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	subnetAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	nicAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	vmImageAccessAPIBehaviorSpec *fakes.APIBehaviorSpec,
	mktPlaceAgreementAccessAPIBehaviorSpec *fakes.APIBehaviorSpec) *fakes.Factory {

	factory := fakes.NewFactory(resourceGroup)
	vmAccess, err := factory.NewVirtualMachineAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(vmAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	vmImageAccess, err := factory.NewImageAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(vmImageAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	subnetAccess, err := factory.NewSubnetAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(subnetAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	mktPlaceAgreementAccess, err := factory.NewMarketPlaceAgreementAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(mktPlaceAgreementAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	nicAccess, err := factory.NewNICAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(nicAccessAPIBehaviorSpec).Build()
	g.Expect(err).To(BeNil())
	diskAccess, err := factory.NewDiskAccessBuilder().WithClusterState(clusterState).Build()
	g.Expect(err).To(BeNil())
	factory.
		WithVirtualMachineAccess(vmAccess).
		WithVirtualMachineImagesAccess(vmImageAccess).
		WithSubnetAccess(subnetAccess).
		WithMarketPlaceAgreementsAccess(mktPlaceAgreementAccess).
		WithNetworkInterfacesAccess(nicAccess).
		WithDisksAccess(diskAccess)

	return factory
}

func getVMNamesFromListMachineResponse(response *driver.ListMachinesResponse) []string {
	if response == nil {
		return []string{}
	}
	vmNames := make([]string, 0, len(response.MachineList))
	for _, vmName := range response.MachineList {
		vmNames = append(vmNames, vmName)
	}
	return vmNames
}

func checkAndGetWrapperAzResponseError(g *WithT, err error, expectedStatusCode codes.Code) *azcore.ResponseError {
	var statusErr *status.Status
	g.Expect(errors.As(err, &statusErr)).To(BeTrue())
	g.Expect(statusErr.Code(), expectedStatusCode)
	cause := statusErr.Cause()
	var azErr *azcore.ResponseError
	g.Expect(errors.As(cause, &azErr)).To(BeTrue())
	return azErr
}
