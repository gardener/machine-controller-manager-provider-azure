// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp/fakes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	corev1 "k8s.io/api/core/v1"
	"testing"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	. "github.com/onsi/gomega"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
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

func TestCreateVM(t *testing.T) {
	const (
		testResourceGroupName = "test-rg"
		testShootNs           = "test-shoot-ns"
		testWorkerPool0Name   = "test-worker-pool-0"
		testNicID             = "/subscriptions/sub-id/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic"
	)
	var (
		testInternalServerError = testhelp.InternalServerError("test-error-code")
		testConflictError       = testhelp.ConflictErr("ZonalAllocationFailed")
	)

	table := []struct {
		description            string
		VMNamesForTestSetup    []string
		targetVMName           string
		shouldOperationSucceed bool
		vmAccessApiBehavior    *fakes.APIBehaviorSpec
		nicAccessApiBehavior   *fakes.APIBehaviorSpec
		diskAccessApiBehavior  *fakes.APIBehaviorSpec
		checkErrorFn           func(g *WithT, err error)
	}{
		{
			description:            "should successfully create a VM",
			VMNamesForTestSetup:    []string{"vm-1"},
			targetVMName:           "vm-1",
			shouldOperationSucceed: true,
			vmAccessApiBehavior:    nil,
			nicAccessApiBehavior:   nil,
			diskAccessApiBehavior:  nil,
		},
		{
			description:            "should return error when VM creation returns back an error that is not ZonalAllocationFailed",
			VMNamesForTestSetup:    []string{"vm-1"},
			targetVMName:           "vm-1",
			shouldOperationSucceed: false,
			vmAccessApiBehavior: fakes.NewAPIBehaviorSpec().
				AddErrorResourceReaction("vm-1", testhelp.AccessMethodBeginCreateOrUpdate, testInternalServerError),
			checkErrorFn: func(g *WithT, err error) {
				var statusErr *status.Status
				g.Expect(errors.As(err, &statusErr)).To(BeTrue())
				g.Expect(statusErr.Code()).To(Equal(codes.Internal))
				g.Expect(errors.Is(statusErr.Cause(), testInternalServerError)).To(BeTrue())
			},
		},
		{
			description:            "should return error when VM creation has a ResourceExhausted error but VM deletion succeeds",
			VMNamesForTestSetup:    []string{"vm-1"},
			targetVMName:           "vm-1",
			shouldOperationSucceed: false,
			vmAccessApiBehavior: fakes.NewAPIBehaviorSpec().
				AddErrorResourceReaction("vm-1", testhelp.AccessMethodBeginCreateOrUpdate, testConflictError),
			checkErrorFn: func(g *WithT, err error) {
				var statusErr *status.Status
				g.Expect(errors.As(err, &statusErr)).To(BeTrue())
				g.Expect(statusErr.Code()).To(Equal(codes.ResourceExhausted))
				g.Expect(errors.Is(statusErr.Cause(), testConflictError)).To(BeTrue())
				g.Expect(errors.Is(statusErr.Cause(), testInternalServerError)).NotTo(BeTrue())
			},
		},
		{
			description:            "should return joined error when VM Creation has a ResourceExhausted error and VM deletion also has an error",
			VMNamesForTestSetup:    []string{"vm-1"},
			targetVMName:           "vm-1",
			shouldOperationSucceed: false,
			vmAccessApiBehavior: fakes.NewAPIBehaviorSpec().
				AddErrorResourceReaction("vm-1", testhelp.AccessMethodBeginCreateOrUpdate, testConflictError).
				AddErrorResourceReaction("vm-1", testhelp.AccessMethodBeginDelete, testInternalServerError),
			checkErrorFn: func(g *WithT, err error) {
				var statusErr *status.Status
				g.Expect(errors.As(err, &statusErr)).To(BeTrue())
				g.Expect(statusErr.Code()).To(Equal(codes.ResourceExhausted))
				g.Expect(errors.Is(statusErr.Cause(), testConflictError)).To(BeTrue())
				g.Expect(errors.Is(statusErr.Cause(), testInternalServerError)).To(BeTrue())
			},
		},
		{
			description:            "should return joined error when BeginCreateOrUpdate has a ResourceExhausted error and NIC deletion has an error, but VM/disk deletion does not have an error",
			VMNamesForTestSetup:    []string{"vm-1"},
			targetVMName:           "vm-1",
			shouldOperationSucceed: false,
			vmAccessApiBehavior: fakes.NewAPIBehaviorSpec().
				AddErrorResourceReaction("vm-1", testhelp.AccessMethodBeginCreateOrUpdate, testConflictError),
			nicAccessApiBehavior: fakes.NewAPIBehaviorSpec().
				AddErrorResourceReaction("vm-1-nic", testhelp.AccessMethodBeginDelete, testInternalServerError),
			checkErrorFn: func(g *WithT, err error) {
				var statusErr *status.Status
				g.Expect(errors.As(err, &statusErr)).To(BeTrue())
				g.Expect(statusErr.Code()).To(Equal(codes.ResourceExhausted))
				g.Expect(errors.Is(statusErr.Cause(), testConflictError)).To(BeTrue())
				g.Expect(err.Error()).To(ContainSubstring("Errors during deletion of NIC/Disks associated to VM"))
			},
		},
		{
			description:            "should return joined error when BeginCreateOrUpdate has a ResourceExhausted error and disk deletion has an error, but VM/nic deletion does not have an error",
			VMNamesForTestSetup:    []string{"vm-1"},
			targetVMName:           "vm-1",
			shouldOperationSucceed: false,
			vmAccessApiBehavior: fakes.NewAPIBehaviorSpec().
				AddErrorResourceReaction("vm-1", testhelp.AccessMethodBeginCreateOrUpdate, testConflictError),
			diskAccessApiBehavior: fakes.NewAPIBehaviorSpec().
				AddErrorResourceReaction("vm-1-os-disk", testhelp.AccessMethodBeginDelete, testInternalServerError),
			checkErrorFn: func(g *WithT, err error) {
				var statusErr *status.Status
				g.Expect(errors.As(err, &statusErr)).To(BeTrue())
				g.Expect(statusErr.Code()).To(Equal(codes.ResourceExhausted))
				g.Expect(errors.Is(statusErr.Cause(), testConflictError)).To(BeTrue())
				g.Expect(err.Error()).To(ContainSubstring("Errors during deletion of NIC/Disks associated to VM"))
			},
		},
	}

	g := NewWithT(t)

	for _, entry := range table {
		t.Run(entry.description, func(_ *testing.T) {
			// Build Provider Spec
			providerSpec := testhelp.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).WithDefaultValues().Build()

			// Create cluster state
			clusterState := fakes.NewClusterState(providerSpec)

			for _, vmName := range entry.VMNamesForTestSetup {
				clusterState.AddMachineResources(fakes.NewMachineResourcesBuilder(providerSpec, vmName).BuildAllResources())
			}

			// create fake factory and initialize vmAccess
			fakeFactory := fakes.NewFactory(testResourceGroupName)
			vmAccess, err := fakeFactory.NewVirtualMachineAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(entry.vmAccessApiBehavior).Build()
			g.Expect(err).To(BeNil())
			nicAccess, err := fakeFactory.NewNICAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(entry.nicAccessApiBehavior).Build()
			g.Expect(err).To(BeNil())
			diskAccess, err := fakeFactory.NewDiskAccessBuilder().WithClusterState(clusterState).WithAPIBehaviorSpec(entry.diskAccessApiBehavior).Build()
			g.Expect(err).To(BeNil())

			fakeFactory.WithVirtualMachineAccess(vmAccess)
			fakeFactory.WithNetworkInterfacesAccess(nicAccess)
			fakeFactory.WithDisksAccess(diskAccess)

			imageRefDiskIDs := make(map[DataDiskLun]DiskID)

			// Call the function
			ctx := context.Background()
			vm, err := CreateVM(ctx, fakeFactory, access.ConnectConfig{}, providerSpec, armcompute.ImageReference{}, nil, &corev1.Secret{}, testNicID, entry.targetVMName, imageRefDiskIDs)

			// Verify results
			if entry.shouldOperationSucceed {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(vm).NotTo(BeNil())
				g.Expect(*vm.Name).To(Equal(entry.targetVMName))
			} else {
				g.Expect(err).To(HaveOccurred())
				g.Expect(vm).To(BeNil())
			}

			if entry.checkErrorFn != nil {
				entry.checkErrorFn(g, err)
			}
		})
	}
}
