package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/sets"

	"golang.org/x/exp/slices"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	fakecompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	fakeresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/fake"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
)

const testdataPath = "testdata"

func TestDriverProviderDeleteMachineSimple(t *testing.T) {
	g := NewWithT(t)
	testMachine, testVM := createTestMachineAndVM(0)
	vmClient, vmClientOptions := createVMClientAndOptionsToFakeServer(t, testVM)
	ctx := context.Background()

	// 1. first get the VM from the VM Client.
	resp, err := vmClient.Get(ctx, "test", testMachine.Name, nil)
	g.Expect(err).To(BeNil())
	t.Logf("(TestDriverProviderDeleteMachineSimple) VM exists with ID: %v", *resp.ID)
	clientProvider := access.NewClientsProviderWithOptions(vmClientOptions, getFakeTokenCredentials)
	driverProvider := NewDefaultDriverWithBehavior(clientProvider, BehaviorOptions{SkipResourceGroupClientAccess: true})

	// 2. Delete the machine using MCM Default Driver.
	machineClass, err := createMachineClass()
	g.Expect(err).To(BeNil())

	_, err = driverProvider.DeleteMachine(ctx, &driver.DeleteMachineRequest{
		Machine:      &testMachine,
		MachineClass: machineClass,
		Secret:       createProviderSecret(),
	})
	g.Expect(err).To(BeNil())

	// 3. Check that there is no machine using the VM access.
	_, err = vmClient.Get(ctx, "test", *testVM.Name, nil)
	g.Expect(err).ToNot(BeNil())
	g.Expect(errors.IsNotFoundAzAPIError(err)).To(BeTrue())
}

func createVMClientAndOptionsToFakeServer(t *testing.T, testVMs ...armcompute.VirtualMachine) (*armcompute.VirtualMachinesClient, *arm.ClientOptions) {
	g := NewWithT(t)
	fakeVMServer := createFakeVMServer(t, testVMs)
	vmClientOptions := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fakecompute.NewVirtualMachinesServerTransport(&fakeVMServer),
		},
	}
	vmClient, err := armcompute.NewVirtualMachinesClient("subscriptionID", azfake.NewTokenCredential(), vmClientOptions)
	g.Expect(err).To(BeNil())
	return vmClient, vmClientOptions
}

func getFakeTokenCredentials(_ access.ConnectConfig) (azcore.TokenCredential, error) {
	return azfake.NewTokenCredential(), nil
}

func createFakeVMServer(t *testing.T, testVMs []armcompute.VirtualMachine) fakecompute.VirtualMachinesServer {
	var vmap = make(map[string]armcompute.VirtualMachine, len(testVMs))
	for _, v := range testVMs {
		vmap[*v.Name] = v
	}
	return fakecompute.VirtualMachinesServer{
		Get: func(ctx context.Context, resourceGroupName, name string, options *armcompute.VirtualMachinesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachinesClientGetResponse], errResp azfake.ErrorResponder) {
			vm, ok := vmap[name]
			if !ok {
				errResp.SetResponseError(notFoundStatus(name))
				return
			}
			vmResp := armcompute.VirtualMachinesClientGetResponse{VirtualMachine: vm}
			resp.SetResponse(http.StatusOK, vmResp, nil)
			return
		},
		BeginDelete: func(ctx context.Context, resourceGroupName string, name string, options *armcompute.VirtualMachinesClientBeginDeleteOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientDeleteResponse], errResp azfake.ErrorResponder) {
			_, ok := vmap[name]
			if !ok {
				errResp.SetResponseError(notFoundStatus(name))
				return
			}
			delete(vmap, name)
			resp.SetTerminalResponse(200, armcompute.VirtualMachinesClientDeleteResponse{}, nil)
			t.Logf("(TestDriverProviderDeleteMachineSimple) fake server: Deleted VM with name: %s", name)
			return
		},
	}
}

func createFakeResourceGroupServer(ctx context.Context, resourceGroupNames sets.Set[string]) fakeresources.ResourceGroupsServer {
	return fakeresources.ResourceGroupsServer{
		CheckExistence: func(ctx context.Context, resourceGroupName string, options *armresources.ResourceGroupsClientCheckExistenceOptions) (resp azfake.Responder[armresources.ResourceGroupsClientCheckExistenceResponse], errResp azfake.ErrorResponder) {
			if !resourceGroupNames.Has(resourceGroupName) {
				errResp.SetResponseError(notFoundStatus(resourceGroupName))
				return
			}
			resourceGroupResp := armresources.ResourceGroupsClientCheckExistenceResponse{Success: true}
			resp.SetResponse(http.StatusOK, resourceGroupResp, nil)
			return
		},
	}
}

func notFoundStatus(name string) (status int, statusCode string) {
	status = 404
	statusCode = fmt.Sprintf("fakeVMServer: Could not find VM with name: %s", name)
	return
}

func findIndexOfVM(name string, vms []armcompute.VirtualMachine) int {
	return slices.IndexFunc(vms, func(m armcompute.VirtualMachine) bool {
		return *m.Name == name
	})
}

func findVMWithName(name string, vms []armcompute.VirtualMachine) (vm armcompute.VirtualMachine, ok bool) {
	idx := findIndexOfVM(name, vms)
	if idx == -1 {
		return
	}
	ok = true
	vm = vms[idx]
	return
}
func createTestMachineAndVM(index int) (v1alpha1.Machine, armcompute.VirtualMachine) {
	machine := v1alpha1.Machine{
		ObjectMeta: newMachineObjectMeta("test", 0),
	}
	vm := armcompute.VirtualMachine{
		Location: to.Ptr("test-location"),
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
	providerSpecPath := filepath.Join(testdataPath, "providerspec.json")
	providerSpecBytes, err := os.ReadFile(providerSpecPath)
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
