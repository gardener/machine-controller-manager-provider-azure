// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	v1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/provider"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/provider/helpers"
)

func wrapAndLogError(errFmt string, err error) error {
	err = fmt.Errorf(errFmt, err)
	fmt.Print("%s,", err)
	return err
}

// deleteDisk deletes the specified disk on Azure
func deleteDisk(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup, diskID string) error {
	var errFmt = "delete operation failed on disk %s with error: %w"
	disksAccess, err := factory.GetDisksAccess(connectConfig)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	delPoller, err := disksAccess.BeginDelete(ctx, resourceGroup, diskID, nil)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	_, err = delPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	fmt.Printf("Deleted an orphaned Disk %s,", diskID)
	return nil
}

// deleteNICs deletes the specified NICs on Azure
func deleteNICs(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup, networkInterfaceName string) error {
	var errFmt = "delete operation failed on NIC %s with error: %w"
	netAccess, err := factory.GetNetworkInterfacesAccess(connectConfig)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	delPoller, err := netAccess.BeginDelete(ctx, resourceGroup, networkInterfaceName, nil)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	_, err = delPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	fmt.Printf("Deleted an orphaned NIC %s,", networkInterfaceName)
	return nil
}

// deleteVM deletes the specified Virtual Machine on Azure
func deleteVM(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup, vmName string) error {
	var errFmt = "delete operation failed on VM %s with error: %w"
	vmAccess, err := factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	delPoller, err := vmAccess.BeginDelete(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	_, err = delPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return wrapAndLogError(errFmt, err)
	}
	fmt.Printf("Deleted an orphan VM %s,", vmName)
	return nil
}

func getAccessFactoryAndConfig(secretData map[string][]byte) (access.Factory, access.ConnectConfig) {
	subscriptionID := helpers.ExtractCredentialsFromData(secretData, api.SubscriptionID, api.AzureSubscriptionID)
	tenantID := helpers.ExtractCredentialsFromData(secretData, api.TenantID, api.AzureTenantID)
	clientID := helpers.ExtractCredentialsFromData(secretData, api.ClientID, api.AzureClientID)
	clientSecret := helpers.ExtractCredentialsFromData(secretData, api.ClientSecret, api.AzureClientSecret)
	connectConfig := access.ConnectConfig{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}
	return access.NewDefaultAccessFactory(), connectConfig
}

// getMachines returns the list of names of the machine objects in the control cluster.
func getMachines(ctx context.Context, factory access.Factory, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	var (
		machines []string
	)
	defaultDriver := provider.NewDefaultDriver(factory)
	machineList, err := defaultDriver.ListMachines(ctx, &driver.ListMachinesRequest{
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

func getAllDisksForResourceGroup(ctx context.Context, resourceGroup string, disksClient *armcompute.DisksClient) ([]*armcompute.Disk, error) {
	var disks []*armcompute.Disk
	for pager := disksClient.NewListByResourceGroupPager(resourceGroup, nil); pager.More(); {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		pageDisks := page.DiskList.Value
		disks = append(disks, pageDisks...)
	}
	return disks, nil
}

// getOrphanedDisks returns the list of orphaned disks.
func getOrphanedDisks(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup string) ([]string, error) {
	disksClient, err := factory.GetDisksAccess(connectConfig)
	if err != nil {
		return nil, err
	}

	var orphanedDisks []string
	disks, err := getAllDisksForResourceGroup(ctx, resourceGroup, disksClient)
	if err != nil {
		return orphanedDisks, err
	}

	for _, disk := range disks {
		if value, ok := disk.Tags[ITResourceTagKey]; ok && *value == ITResourceTagValue {
			orphanedDisks = append(orphanedDisks, *disk.Name)
		}
	}

	return orphanedDisks, nil
}

func getAllNetInterfacesForResourceGroup(ctx context.Context, resourceGroup string, nifClient *armnetwork.InterfacesClient) ([]*armnetwork.Interface, error) {
	var netIfs []*armnetwork.Interface
	for pager := nifClient.NewListPager(resourceGroup, nil); pager.More(); {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		pageNetIfs := page.InterfaceListResult.Value
		netIfs = append(netIfs, pageNetIfs...)
	}
	return netIfs, nil
}

// getOrphanedNICs returns the list of orphaned NICs.
func getOrphanedNICs(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup string) ([]string, error) {
	var errFmt = "list operation failed on NIC from resource group %s with error %w"
	var orphanedNICs []string
	nifClient, err := factory.GetNetworkInterfacesAccess(connectConfig)
	if err != nil {
		return nil, wrapAndLogError(errFmt, err)
	}

	networkInterfaces, err := getAllNetInterfacesForResourceGroup(ctx, resourceGroup, nifClient)
	if err != nil {
		return nil, wrapAndLogError(errFmt, err)
	}

	for _, networkInterface := range networkInterfaces {
		if value, ok := networkInterface.Tags[ITResourceTagKey]; ok && ITResourceTagValue == *value {
			orphanedNICs = append(orphanedNICs, *networkInterface.Name)
		}
	}
	return orphanedNICs, nil
}

func getAllVmsForResourceGroup(ctx context.Context, resourceGroup string, vmClient *armcompute.VirtualMachinesClient) ([]*armcompute.VirtualMachine, error) {
	var vms []*armcompute.VirtualMachine
	for pager := vmClient.NewListPager(resourceGroup, nil); pager.More(); {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		pageVms := page.VirtualMachineListResult.Value
		vms = append(vms, pageVms...)
	}
	return vms, nil
}

// getOrphanedVMs returns the list of orphaned virtual machines.
func getOrphanedVMs(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup string) ([]string, error) {
	vmClient, err := factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return nil, err
	}
	virtualMachines, err := getAllVmsForResourceGroup(ctx, resourceGroup, vmClient)
	if err != nil {
		return nil, err
	}
	var orphanedVMs []string
	for _, virtualMachine := range virtualMachines {
		if value, ok := virtualMachine.Tags[ITResourceTagKey]; ok && *value == ITResourceTagValue {
			orphanedVMs = append(orphanedVMs, *virtualMachine.Name)
		}
	}
	return orphanedVMs, nil
}

func cleanUpOrphanedResources(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup string,
	orphanedVms []string, orphanedVolumes []string, orphanedNICs []string) (delErrOrphanedVms []string, delErrOrphanedVolumes []string, delErrOrphanedNICs []string) {
	for _, virtualMachineName := range orphanedVms {
		err := deleteVM(ctx, factory, connectConfig, resourceGroup, virtualMachineName)
		if err != nil {
			delErrOrphanedVms = append(delErrOrphanedVms, virtualMachineName)
		}
	}
	for _, volumeID := range orphanedVolumes {
		err := deleteDisk(ctx, factory, connectConfig, resourceGroup, volumeID)
		if err != nil {
			delErrOrphanedVolumes = append(delErrOrphanedVolumes, volumeID)
		}
	}
	for _, networkInterfaceName := range orphanedNICs {
		err := deleteNICs(ctx, factory, connectConfig, resourceGroup, networkInterfaceName)
		if err != nil {
			delErrOrphanedNICs = append(delErrOrphanedNICs, networkInterfaceName)
		}
	}
	return
}
