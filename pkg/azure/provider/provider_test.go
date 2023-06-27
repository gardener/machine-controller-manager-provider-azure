package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	driver "github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client"
)

func TestDriverProviderDeleteMachineSimple(t *testing.T) {
	fakeVMServer := fake.VirtualMachinesServer{
		Get: func(ctx context.Context, resourceGroupName, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachinesClientGetResponse], errResp azfake.ErrorResponder) {
			vmResp := armcompute.VirtualMachinesClientGetResponse{}
			vmResp.ID = to.Ptr("/fake/resource/id")
			resp.SetResponse(http.StatusOK, vmResp, nil)
			return
		},

		BeginDelete: func(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientBeginDeleteOptions) (resp azfake.PollerResponder[armcompute.VirtualMachinesClientDeleteResponse], errResp azfake.ErrorResponder) {
			delResp := armcompute.VirtualMachinesClientDeleteResponse{}
			resp.SetTerminalResponse(200, delResp, nil)
			return
		},
	}
	clientOptions := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fake.NewVirtualMachinesServerTransport(&fakeVMServer),
		},
	}
	clientProvider := client.NewClientsProviderWithOptions(clientOptions, azfake.NewTokenCredential())
	driverProvider := NewDriverWithBehavior(clientProvider, BehaviorOptions{SkipResourceGroupClientAccess: true})
	ctx := context.Background()
	machine := &v1alpha1.Machine{
		ObjectMeta: newObjectMeta("test", 0),
	}
	deleteMachineResp, err := driverProvider.DeleteMachine(ctx, &driver.DeleteMachineRequest{
		Machine:      machine,
		MachineClass: createMachineClass(),
		Secret:       createProviderSecret(),
	})
	t.Logf("(TestDriverProviderDeleteMachineSimple) Got delete machine response: %v", deleteMachineResp)
	if err != nil {
		t.Fatalf("Failed to delete machine: %v", err)
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
    "resourceGroup": "<resource-group-name>",
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
