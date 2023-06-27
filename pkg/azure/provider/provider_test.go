package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"golang.org/x/exp/slices"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client"
)

func TestDriverProviderDeleteMachineSimple(t *testing.T) {
	testMachine, testVM := createTestMachineAndVM(0)
	vmClient, vmClientOptions := createVMClientAndOptionsToFakeServer(t, testVM)
	ctx := context.Background()

	// 1. first get the VM from the VM Client.
	resp, err := vmClient.Get(context.TODO(), "test", testMachine.Name, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("(TestDriverProviderDeleteMachineSimple) VM exists with ID: %v", *resp.ID)
	clientProvider := client.NewClientsProviderWithOptions(vmClientOptions, azfake.NewTokenCredential())
	driverProvider := NewDefaultDriverWithBehavior(clientProvider, BehaviorOptions{SkipResourceGroupClientAccess: true})

	// 2. Delete the machine using MCM Default Driver.
	_, err = driverProvider.DeleteMachine(ctx, &driver.DeleteMachineRequest{
		Machine:      &testMachine,
		MachineClass: createMachineClass(),
		Secret:       createProviderSecret(),
	})
	if err != nil {
		t.Fatalf("(TestDriverProviderDeleteMachineSimple) Failed to delete machine: %v", err)
	}

	// 3. Check that there is no machine using the VM client.
	resp, err = vmClient.Get(ctx, "test", *testVM.Name, nil)
	if err == nil {
		t.Error("(TestDriverProviderDeleteMachineSimple) want error for vmClient.Get with vm name:", *testVM.Name)
	}
}

func createVMClientAndOptionsToFakeServer(t *testing.T, testVMs ...armcompute.VirtualMachine) (*armcompute.VirtualMachinesClient, *arm.ClientOptions) {
	fakeVMServer := createFakeVMServer(t, testVMs)
	vmClientOptions := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fake.NewVirtualMachinesServerTransport(&fakeVMServer),
		},
	}
	vmClient, err := armcompute.NewVirtualMachinesClient("subscriptionID", azfake.NewTokenCredential(), vmClientOptions)
	if err != nil {
		t.Fatal(err)
	}
	return vmClient, vmClientOptions
}

func createFakeVMServer(t *testing.T, testVMs []armcompute.VirtualMachine) fake.VirtualMachinesServer {
	var vmap = make(map[string]armcompute.VirtualMachine, len(testVMs))
	for _, v := range testVMs {
		vmap[*v.Name] = v
	}
	return fake.VirtualMachinesServer{
		Get: func(ctx context.Context, resourceGroupName, name string, options *armcompute.VirtualMachinesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachinesClientGetResponse], errResp azfake.ErrorResponder) {
			vm, ok := vmap[name]
			if !ok {
				errResp.SetResponseError(vmNotFoundStatus(name))
				return
			}
			vmResp := armcompute.VirtualMachinesClientGetResponse{vm}
			resp.SetResponse(http.StatusOK, vmResp, nil)
			return
		},
		BeginDelete: func(ctx context.Context, resourceGroupName string, name string, options *armcompute.VirtualMachinesClientBeginDeleteOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientDeleteResponse], errResp azfake.ErrorResponder) {
			_, ok := vmap[name]
			if !ok {
				errResp.SetResponseError(vmNotFoundStatus(name))
				return
			}
			delete(vmap, name)
			resp.SetTerminalResponse(200, armcompute.VirtualMachinesClientDeleteResponse{}, nil)
			t.Logf("(TestDriverProviderDeleteMachineSimple) fake server: Deleted VM with name: %s", name)
			return
		},
	}
}

func vmNotFoundStatus(name string) (status int, statusCode string) {
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
		ObjectMeta: newObjectMeta("test", 0),
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
		//Tags:             nil,
		//Zones:            nil,
		ID:   to.Ptr("test-vm-" + strconv.Itoa(index)),
		Name: to.Ptr(machine.Name),
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
func createMachineClass() *v1alpha1.MachineClass {
	machineClass := &v1alpha1.MachineClass{
		Provider: "Azure",
		ProviderSpec: runtime.RawExtension{
			Raw:    []byte(rawProviderSpec),
			Object: nil,
		},
	}
	return machineClass
}

func newObjectMeta(namespace string, machineIndex int) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{
		GenerateName: "class",
		Namespace:    namespace,
	}
	meta.Name = fmt.Sprintf("machine-%d", machineIndex)
	return meta
}

const rawProviderSpec = `
{
    "location": "westeurope",
    "properties": {
      "hardwareProfile": {
        "vmSize": "Standard_DS2_v2"
      },
      "networkProfile": {
        "networkInterfaces": {},
        "acceleratedNetworking": false
      },
      "osProfile": {
        "adminUsername": "core",
        "customData": "<string>",
        "computerName": "<string>",
        "linuxConfiguration": {
          "disablePasswordAuthentication": true,
          "ssh": {
            "publicKeys": {
              "keyData": "<SSH-RSA KEY>",
              "path": "<path to the rsa-ssh key>"
            }
          }
        }
      },
      "storageProfile": {
        "imageReference": {
          "urn": "sap:gardenlinux:greatest:184.0.0"
        },
        "osDisk": {
          "caching": "None",
          "createOption": "FromImage",
          "diskSizeGB": 50,
          "managedDisk": {
            "storageAccountType": "<eg:Standard_LRS>"
          }
        }
      },
      "zone": 2,
      "identityID": "<string>"
    },
    "resourceGroup": "test",
    "subnetInfo": {
      "subnetName": "<subnet-name>",
      "vnetName": "<vnet-name>"
    },
    "tags": {
      "Name": "<name>",
      "kubernetes.io-cluster-<name>": "1",
      "kubernetes.io-role-node": "1",
      "node.kubernetes.io_role": "node",
      "worker.garden.sapcloud.io_group": "<worker-group-name>",
      "worker.gardener.cloud_pool": "<worker-group-name>",
      "worker.gardener.cloud_system-components": "true"
    }
}`
