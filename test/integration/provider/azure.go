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

	provider "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/spi"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
)

func getAzureClients(secretData map[string][]byte) (spi.AzureDriverClientsInterface, error) {
	var SPI spi.PluginSPIImpl
	driver := provider.NewAzureDriver(&SPI)
	client, err := driver.SPI.Setup(&v1.Secret{Data: secretData})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func DescribeMachines(machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	var (
		machines []string
		SPI      spi.PluginSPIImpl
	)

	driverprovider := provider.NewAzureDriver(&SPI)
	machineList, err := driverprovider.ListMachines(context.TODO(), &driver.ListMachinesRequest{
		MachineClass: machineClass,
		Secret:       &v1.Secret{Data: secretData},
	})

	if err != nil {
		return nil, err
	} else if len(machineList.MachineList) != 0 {
		fmt.Printf("\nAvailable Machines: ")
		for _, machine := range machineList.MachineList {
			machines = append(machines, machine)
		}
	}
	return machines, nil
}

// GetVMsWithTag describes the instance with the specified tag
func GetVMsWithTag(clients spi.AzureDriverClientsInterface, tagName string, tagValue string, machineClass *v1alpha1.MachineClass, resourceGroup string, secretData map[string][]byte) ([]string, error) {

	var instancesID []string

	virtualMachines, err := clients.GetVM().List(context.TODO(), resourceGroup)
	if err != nil {
		return instancesID, err
	}
	for _, virtualMachine := range virtualMachines.Values() {
		if value, ok := virtualMachine.Tags[tagName]; ok && *value == tagValue {
			instancesID = append(instancesID, *virtualMachine.Name)
		}
	}

	return instancesID, nil
}

// GetAvailableDisks describes volumes with the specified tag
func GetAvailableDisks(clients spi.AzureDriverClientsInterface, tagName string, tagValue string, machineClass *v1alpha1.MachineClass, resourceGroup string, secretData map[string][]byte) ([]string, error) {

	var availVolID []string
	// extract the resource group value from the providerSpec of MachineClass

	volumes, err := clients.GetDisk().ListByResourceGroup(context.TODO(), resourceGroup)
	if err != nil {
		return availVolID, err
	}

	for _, volume := range volumes.Values() {
		if value, ok := volume.Tags[tagName]; ok && *value == tagValue {
			availVolID = append(availVolID, *volume.Name)
			DeleteVolume(clients, resourceGroup, *volume.Name)
		}
	}

	return availVolID, nil
}

// DeleteVolume deletes the specified volume
func DeleteVolume(clients spi.AzureDriverClientsInterface, resourceGroup, VolumeID string) error {
	// TO-DO: deletes an available volume with the specified volume ID
	// If the command succeeds, no output is returned.
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

// AdditionalResourcesCheck describes VPCs and network interfaces
func AdditionalResourcesCheck(clients spi.AzureDriverClientsInterface, resourceGroup, tagName, tagValue string) error {
	// TO-DO: Checks for Network interfaces
	// If the command succeeds, no output is returned.
	networkInterfaces, err := clients.GetNic().List(context.TODO(), resourceGroup)
	if err != nil {
		return err
	}
	for _, networkInterface := range networkInterfaces.Values() {
		fmt.Println(networkInterface.Name)
	}
	return err
}
