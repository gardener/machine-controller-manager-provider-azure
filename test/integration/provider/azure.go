package provider

/**
	Orphaned Resources
	- VMs:
		Describe instances with specified tag name:<cluster-name>
		Report/Print out instances found
		Describe volumes attached to the instance (using instance id)
		Report/Print out volumes found
		Delete attached volumes found
		Terminate instances found
	- Disks:
		Describe volumes with tag status:available
		Report/Print out volumes found
		Delete identified volumes
**/

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	provider "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/spi"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
)

// AdditionalResourcesCheck check for orphan network interfaces and triggers their deletion
func additionalResourcesCheck(
	clients spi.AzureDriverClientsInterface,
	resourceGroup,
	tagName,
	tagValue string,
) error {
	ctx := context.TODO()

	networkInterfaces, err := clients.GetNic().List(ctx, resourceGroup)
	if err != nil {
		return err
	}
	for _, networkInterface := range networkInterfaces.Values() {
		if value, ok := networkInterface.Tags[tagName]; ok && tagValue == *value {
			fmt.Println(networkInterface.Name)
			nicDeleteFuture, err := clients.GetNic().Delete(ctx, resourceGroup, *networkInterface.Name)
			if err != nil {
				return err
			}
			if err = nicDeleteFuture.WaitForCompletionRef(ctx, clients.GetClient()); err != nil {
				return err
			}
		}
	}
	return err
}

// deleteVolume deletes the specified volume
func deleteVolume(
	clients spi.AzureDriverClientsInterface,
	resourceGroup,
	VolumeID string,
) error {
	ctx := context.TODO()
	diskDeleteFuture, err := clients.GetDisk().Delete(ctx, resourceGroup, VolumeID)
	if err != nil {
		return err
	}

	if err = diskDeleteFuture.WaitForCompletionRef(ctx, clients.GetClient()); err != nil {
		return err
	}
	return err
}

// getAvailableDisks fetches all the disks associated with the VMs of the target cluster
// subject to integration test and deletes them
func getAvailableDisks(
	clients spi.AzureDriverClientsInterface,
	tagName string,
	tagValue string,
	resourceGroup string,
) ([]string, error) {

	var availVolID []string

	volumes, err := clients.GetDisk().ListByResourceGroup(context.TODO(), resourceGroup)
	if err != nil {
		return availVolID, err
	}

	for _, volume := range volumes.Values() {
		if value, ok := volume.Tags[tagName]; ok && *value == tagValue {
			availVolID = append(availVolID, *volume.Name)
			deleteVolume(clients, resourceGroup, *volume.Name)
		}
	}

	return availVolID, nil
}

// getAzureClients returns Azure clients
func getAzureClients(
	secretData map[string][]byte,
) (spi.AzureDriverClientsInterface, error) {

	driver := provider.NewAzureDriver(&spi.PluginSPIImpl{})
	client, err := driver.SPI.Setup(&v1.Secret{Data: secretData})
	if err != nil {
		return nil, err
	}
	return client, nil
}

// getMachines returns the list of names of the machine objects in the control cluster
func getMachines(
	machineClass *v1alpha1.MachineClass,
	secretData map[string][]byte,
) ([]string, error) {
	var (
		machines []string
		SPI      spi.PluginSPIImpl
	)

	driverprovider := provider.NewAzureDriver(&SPI)
	machineList, err := driverprovider.ListMachines(context.TODO(), &driver.ListMachinesRequest{
		MachineClass: machineClass,
		Secret: &v1.Secret{
			Data: secretData,
		},
	})

	if err != nil {
		return nil, err
	} else if len(machineList.MachineList) != 0 {
		fmt.Printf("\nAvailable Machines: ")
		for _, machine := range machineList.MachineList {
			fmt.Printf("%s,", machine)
			machines = append(machines, machine)
		}
	}
	return machines, nil
}

// getVMsWithTag describes the instance with the specified tag from the target cluster
// and triggers the deletion
func getVMsWithTag(
	clients spi.AzureDriverClientsInterface,
	tagName string,
	tagValue string,
	machineClass *v1alpha1.MachineClass,
	resourceGroup string,
	secretData map[string][]byte,
) ([]string, error) {

	var (
		instancesID []string
		ctx         = context.TODO()
	)

	virtualMachines, err := clients.GetVM().List(ctx, resourceGroup)
	if err != nil {
		return instancesID, err
	}

	for _, virtualMachine := range virtualMachines.Values() {
		if value, ok := virtualMachine.Tags[tagName]; ok && *value == tagValue {
			instancesID = append(instancesID, *virtualMachine.Name)
			virtualMachineFuture, err := clients.GetVM().Delete(ctx, resourceGroup, *virtualMachine.Name, pointer.BoolPtr(false))
			if err != nil {
				fmt.Printf("Delete operation failed on VM %s with error %s,", *virtualMachine.Name, err.Error())
			}

			if err = virtualMachineFuture.WaitForCompletionRef(ctx, clients.GetClient()); err != nil {
				fmt.Printf("Delete operation failed on VM %s with error %s,", *virtualMachine.Name, err.Error())
			}
		}
	}

	return instancesID, nil
}
