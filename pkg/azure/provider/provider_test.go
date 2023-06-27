package provider

import (
	"context"
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
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client"
)

func TestDriverProvider_DeleteMachine_Simple(t *testing.T) {
	fakeVMServer := fake.VirtualMachinesServer{
		// next, provide implementations for the APIs you wish to fake.
		// this fake corresponds to the VirtualMachinesClient.Get() API.
		Get: func(ctx context.Context, resourceGroupName, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (resp azfake.Responder[armcompute.VirtualMachinesClientGetResponse], errResp azfake.ErrorResponder) {
			// the values of ctx, resourceGroupName, vmName, and options come from the API call.

			// the named return values resp and errResp are used to construct the response
			// and are meant to be mutually exclusive. if both responses have been constructed,
			// the error response is selected.

			// construct the response type, populating fields as required
			vmResp := armcompute.VirtualMachinesClientGetResponse{}
			vmResp.ID = to.Ptr("/fake/resource/id")

			// use resp to set the desired response
			resp.SetResponse(http.StatusOK, vmResp, nil)

			// to simulate the failure case, use errResp
			//errResp.SetResponseError(http.StatusBadRequest, "ThisIsASimulatedError")

			return
		},
	}
	clientOptions := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fake.NewVirtualMachinesServerTransport(&fakeVMServer),
		},
	}
	clientProvider := client.NewClientsProviderWithOptions(clientOptions, azfake.NewTokenCredential())
	driverProvider := NewDriver(clientProvider)
	ctx := context.Background()
	machine := &v1alpha1.Machine{}
	secret := &corev1.Secret{}
	deleteMachineResp, err := driverProvider.DeleteMachine(ctx, &driver.DeleteMachineRequest{
		Machine:      machine,
		MachineClass: createMachineClass(),
		Secret:       secret,
	})
	t.Logf("Got delete machine response: %v", deleteMachineResp)
	t.Fatalf("Failed to delete machine: %v", err)
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

const rawProviderSpec = `
{
    "location": "westeurope",
    "properties": {
      "hardwareProfile": {
        "vmSize": "Standard_DS2_v2"
      },
      "networkProfile": {
        "networkInterfaces": {},
        "acceleratedNetworking": "<boolean>"
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
      "identityID": "<string>",
      "availabilitySet": {
        "id": "<string>"
      },
      "machineSet": {
        "id": "<string>",
        "Kind": "<string>"
      }
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

/*
{
  "apiVersion": "machine.sapcloud.io/v1alpha1",
  "credentialsSecretRef": {
    "name": "<secret-name>",
    "namespace": "<namespace-of-secret>"
  },
  "kind": "MachineClass",
  "metadata": {
    "name": "<machineclass-name>",
    "namespace": "<machineclass namespace>"
  },
  "provider": "Azure",
  "providerSpec": {
    "location": "westeurope",
    "properties": {
      "hardwareProfile": {
        "vmSize": "Standard_DS2_v2"
      },
      "networkProfile": {
        "networkInterfaces": {},
        "acceleratedNetworking": "<boolean>"
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
      "identityID": "<string>",
      "availabilitySet": {
        "id": "<string>"
      },
      "machineSet": {
        "id": "<string>",
        "Kind": "<string>"
      }
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
  },
  "secretRef": {
    "name": "<secret-name>",
    "namespace": "<namespace-of-secret>"
  }
}
*/
