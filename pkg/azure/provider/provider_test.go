package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/test/fakes"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
)

const (
	testLocation          = "westeurope"
	testResourceGroupName = "test-rg"
	testShootNs           = "test-shoot-ns"
	testWorkerPool0Name   = "test-worker-pool-0"
)

func TestDeleteMachine(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	testMachine, testVM := createTestMachineAndVM(0)

	fakeFactory := fakes.NewFactory(testResourceGroupName)
	vmAccess, err := fakeFactory.NewVirtualMachineAccessBuilder().
		WithExistingVMs([]armcompute.VirtualMachine{testVM}).
		WithGet(nil).
		WithBeginDelete(nil).
		Build()
	g.Expect(err).To(BeNil())
	rgAccess, err := fakeFactory.NewResourceGroupsAccessBuilder().WithCheckExistence(nil).Build()
	g.Expect(err).To(BeNil())
	fakeFactory.WithVirtualMachineAccess(vmAccess).WithResourceGroupsAccess(rgAccess)

	// 1. first get the VM from the VM Client.
	resp, err := vmAccess.Get(ctx, testResourceGroupName, testMachine.Name, nil)
	g.Expect(err).To(BeNil())
	t.Logf("(TestDriverProviderDeleteMachineSimple) VM exists with ID: %v", *resp.ID)

	testDriver := NewDefaultDriver(fakeFactory)

	// 2. Delete the machine using MCM Default Driver.
	machineClass, err := createMachineClass()
	g.Expect(err).To(BeNil())

	_, err = testDriver.DeleteMachine(ctx, &driver.DeleteMachineRequest{
		Machine:      &testMachine,
		MachineClass: machineClass,
		Secret:       createProviderSecret(),
	})
	g.Expect(err).To(BeNil())

	// 3. Check that there is no machine using the VM access.
	_, err = vmAccess.Get(ctx, testResourceGroupName, *testVM.Name, nil)
	g.Expect(err).ToNot(BeNil())
	g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
}

func createTestMachineAndVM(index int) (v1alpha1.Machine, armcompute.VirtualMachine) {
	machine := v1alpha1.Machine{
		ObjectMeta: newMachineObjectMeta(testShootNs, 0),
	}
	vm := armcompute.VirtualMachine{
		Location: to.Ptr(testLocation),
		//Identity:         nil,
		//Plan:             nil,
		Properties: &armcompute.VirtualMachineProperties{
			NetworkProfile: createNetworkProfile(index),
			//OSProfile:               nil,
			//PlatformFaultDomain:     nil,
			//Priority:                nil,
			//ProximityPlacementGroup: nil,
			//ScheduledEventsProfile:  nil,
			//SecurityProfile:         nil,
			//StorageProfile:          nil,
			//UserData:                nil,
			//VirtualMachineScaleSet:  nil,
			//InstanceView:            nil,
			//ProvisioningState:       nil,
			//VMID:                    nil,
		},
		ID:   to.Ptr("test-vm-" + strconv.Itoa(index)),
		Name: to.Ptr(machine.Name),
		//Tags:             nil,
		//Zones:            nil,
		//Resources:        nil,
		//Type:             nil,
	}
	return machine, vm
}

func createNetworkProfile(index int) *armcompute.NetworkProfile {
	var networkIfName = "test-netif-" + strconv.Itoa(index)
	return &armcompute.NetworkProfile{
		NetworkInterfaceConfigurations: nil,
		NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
			{
				ID:         to.Ptr(networkIfName),
				Properties: nil,
			},
		},
	}
}

func createProviderSecret() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			api.ClientID:       []byte("test"),
			api.ClientSecret:   []byte("test"),
			api.SubscriptionID: []byte("test"),
			api.TenantID:       []byte("test"),
		},
	}
}
func createMachineClass() (*v1alpha1.MachineClass, error) {
	providerSpecBytes, err := test.NewProviderSpecBuilder(testResourceGroupName, testShootNs, testWorkerPool0Name).
		WithDefaultValues().
		Marshal()
	if err != nil {
		return nil, err
	}

	machineClass := &v1alpha1.MachineClass{
		Provider: "Azure",
		ProviderSpec: runtime.RawExtension{
			Raw:    providerSpecBytes,
			Object: nil,
		},
	}
	return machineClass, nil
}

func newMachineObjectMeta(namespace string, machineIndex int) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      fmt.Sprintf("machine-%d", machineIndex),
		Namespace: namespace,
	}
}
