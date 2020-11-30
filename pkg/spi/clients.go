/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package spi implements the helper or auxilliary methods for AzureDriverClient
package spi

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/marketplaceordering/mgmt/marketplaceordering"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	computeapi "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute/computeapi"
	marketplaceorderingapi "github.com/Azure/azure-sdk-for-go/services/marketplaceordering/mgmt/2015-06-01/marketplaceordering/marketplaceorderingapi"
	networkapi "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-04-01/network/networkapi"
	"github.com/Azure/go-autorest/autorest"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog"
)

const (
	prometheusServiceSubnet = "subnet"
	prometheusServiceVM     = "virtual_machine"
	prometheusServiceNIC    = "network_interfaces"
	prometheusServiceDisk   = "disks"
)

// AzureDriverClientsInterface is the interfaces to be implemented
// by the AzureDriverClients to get and refer the respective clients
type AzureDriverClientsInterface interface {

	// GetSubnet() is the getter for the Azure Subnets Client
	GetSubnet() networkapi.SubnetsClientAPI

	// GetNic() is the getter for the Azure Interfaces Client
	GetNic() networkapi.InterfacesClientAPI

	// GetVM() is the getter for the Azure Virtual Machines Client
	GetVM() computeapi.VirtualMachinesClientAPI

	// GetDisk() is the getter for the Azure Disks Client
	GetDisk() computeapi.DisksClientAPI

	// GetImages() is the getter for the Azure Virtual Machines Images Client
	GetImages() computeapi.VirtualMachineImagesClientAPI

	// GetDeployments() is the getter for the Azure Deployment Client
	GetDeployments() resources.DeploymentsClient

	// GetMarketplace() is the getter for the Azure Marketplace Agreement Client
	GetMarketplace() marketplaceorderingapi.MarketplaceAgreementsClientAPI

	// GetClient() is the getter of the Azure autorest client
	GetClient() autorest.Client
}

// azureDriverClients . . .
type azureDriverClients struct {
	subnet      network.SubnetsClient
	nic         network.InterfacesClient
	vm          compute.VirtualMachinesClient
	disk        compute.DisksClient
	deployments resources.DeploymentsClient
	images      compute.VirtualMachineImagesClient
	group       resources.GroupsClient
	marketplace marketplaceordering.MarketplaceAgreementsClient
}

// GetVM method is the getter for the Virtual Machines Client from the AzureDriverClients
func (clients azureDriverClients) GetVM() compute.VirtualMachinesClient {
	return clients.vm
}

// GetDisk method is the getter for the Disks Client from the AzureDriverClients
func (clients azureDriverClients) GetDisk() compute.DisksClient {
	return clients.disk
}

// GetImages is the getter for the Virtual Machines Images Client from the AzureDriverClients
func (clients azureDriverClients) GetImages() compute.VirtualMachineImagesClient {
	return clients.images
}

// GetNic is the getter for the  Network Interfaces Client from the AzureDriverClients
func (clients azureDriverClients) GetNic() network.InterfacesClient {
	return clients.nic
}

// GetSubnet is the getter for the Network Subnets Client from the AzureDriverClients
func (clients azureDriverClients) GetSubnet() network.SubnetsClient {
	return clients.subnet
}

// GetDeployments is the getter for the resources deployment from the AzureDriverClients
func (clients azureDriverClients) GetDeployments() resources.DeploymentsClient {
	return clients.deployments
}

// GetGroup is the getter for the resources Group Client from the AzureDriverClients
func (clients azureDriverClients) GetGroup() resources.GroupsClient {
	return clients.group
}

// GetMarketplace is the getter for the marketplace agreement client from the AzureDriverClients
func (clients azureDriverClients) GetMarketplace() marketplaceordering.MarketplaceAgreementsClient {
	return clients.marketplace
}

// GetClient is the getter for the autorest Client from the AzureDriverClients
func (clients azureDriverClients) GetClient() autorest.Client {
	return clients.vm.BaseClient.Client
}

func (clients *azureDriverClients) DeleteVM(ctx context.Context, resourceGroupName string, vmName string) error {
	klog.V(2).Infof("VM deletion has began for %q", vmName)
	defer klog.V(2).Infof("VM deleted for %q", vmName)

	future, err := clients.GetVM().Delete(ctx, resourceGroupName, vmName)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceVM, err, "vm.Delete")
	}
	err = future.WaitForCompletionRef(ctx, clients.GetVM().Client)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceVM, err, "vm.Delete")
	}
	OnARMAPISuccess(prometheusServiceVM, "VM deletion was successful for %s", vmName)
	return nil
}

func (clients *azureDriverClients) WaitForDataDiskDetachment(ctx context.Context, resourceGroupName string, vm compute.VirtualMachine) error {
	klog.V(2).Infof("Data disk detachment began for %q", *vm.Name)
	defer klog.V(2).Infof("Data disk detached for %q", *vm.Name)

	if len(*vm.StorageProfile.DataDisks) > 0 {
		// There are disks attached hence need to detach them
		vm.StorageProfile.DataDisks = &[]compute.DataDisk{}

		future, err := clients.GetVM().CreateOrUpdate(ctx, resourceGroupName, *vm.Name, vm)
		if err != nil {
			return OnARMAPIErrorFail(prometheusServiceVM, err, "Failed to CreateOrUpdate. Error Message - %s", err)
		}
		err = future.WaitForCompletionRef(ctx, clients.GetVM().Client)
		if err != nil {
			return OnARMAPIErrorFail(prometheusServiceVM, err, "Failed to CreateOrUpdate. Error Message - %s", err)
		}
		OnARMAPISuccess(prometheusServiceVM, "VM CreateOrUpdate was successful for %s", *vm.Name)
	}

	return nil
}

func (clients *azureDriverClients) FetchAttachedVMfromNIC(ctx context.Context, resourceGroupName, nicName string) (string, error) {
	nic, err := clients.GetNic().Get(ctx, resourceGroupName, nicName, "")
	if err != nil {
		return "", err
	}
	if nic.VirtualMachine == nil {
		return "", nil
	}
	return *nic.VirtualMachine.ID, nil
}

func (clients *azureDriverClients) DeleteNIC(ctx context.Context, resourceGroupName string, nicName string) error {
	klog.V(2).Infof("NIC delete started for %q", nicName)
	defer klog.V(2).Infof("NIC deleted for %q", nicName)

	future, err := clients.GetNic().Delete(ctx, resourceGroupName, nicName)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.Delete")
	}
	if err := future.WaitForCompletionRef(ctx, clients.GetNic().Client); err != nil {
		return OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.Delete")
	}
	OnARMAPISuccess(prometheusServiceNIC, "NIC deletion was successful for %s", nicName)
	return nil
}

func (clients *azureDriverClients) fetchAttachedVMfromDisk(ctx context.Context, resourceGroupName, diskName string) (string, error) {
	disk, err := clients.GetDisk().Get(ctx, resourceGroupName, diskName)
	if err != nil {
		return "", err
	}
	if disk.ManagedBy == nil {
		return "", nil
	}
	return *disk.ManagedBy, nil
}

func (clients *azureDriverClients) deleteDisk(ctx context.Context, resourceGroupName string, diskName string) error {
	klog.V(2).Infof("Disk delete started for %q", diskName)
	defer klog.V(2).Infof("Disk deleted for %q", diskName)

	future, err := clients.GetDisk().Delete(ctx, resourceGroupName, diskName)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceDisk, err, "disk.Delete")
	}
	if err = future.WaitForCompletionRef(ctx, clients.GetDisk().Client); err != nil {
		return OnARMAPIErrorFail(prometheusServiceDisk, err, "disk.Delete")
	}
	OnARMAPISuccess(prometheusServiceDisk, "Disk deletion was successful for %s", diskName)
	return nil
}

func (clients *azureDriverClients) GetDeleterForDisk(ctx context.Context, resourceGroupName string, diskName string) func() error {
	return func() error {
		if vmHoldingDisk, err := clients.fetchAttachedVMfromDisk(ctx, resourceGroupName, diskName); err != nil {
			if NotFound(err) {
				// Resource doesn't exist, no need to delete
				return nil
			}
			return err
		} else if vmHoldingDisk != "" {
			return fmt.Errorf("Cannot delete disk %s because it is attached to VM %s", diskName, vmHoldingDisk)
		}

		return clients.deleteDisk(ctx, resourceGroupName, diskName)
	}
}

// PrometheusFail ...
func PrometheusFail(service string) {
	metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "azure", "service": service}).Inc()
}

// PrometheusSuccess ..
func PrometheusSuccess(service string) {
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "azure", "service": service}).Inc()
}

// RetrieveRequestID ...
func RetrieveRequestID(err error) (bool, string, *autorest.DetailedError) {
	switch err.(type) {
	case autorest.DetailedError:
		detailedErr := autorest.DetailedError(err.(autorest.DetailedError))
		if detailedErr.Response != nil {
			requestID := strings.Join(detailedErr.Response.Header["X-Ms-Request-Id"], "")
			return true, requestID, &detailedErr
		}
		return false, "", nil
	default:
		return false, "", nil
	}
}

// OnErrorFail prints a failure message and exits the program if err is not nil.
func OnErrorFail(err error, format string, v ...interface{}) error {
	if err != nil {
		message := fmt.Sprintf(format, v...)
		if hasRequestID, requestID, detailedErr := RetrieveRequestID(err); hasRequestID {
			klog.Errorf("Azure ARM API call with x-ms-request-id=%s failed. %s: %s\n", requestID, message, *detailedErr)
		} else {
			klog.Errorf("%s: %s\n", message, err)
		}
	}
	return err
}

// OnARMAPIErrorFail ...
func OnARMAPIErrorFail(prometheusService string, err error, format string, v ...interface{}) error {
	PrometheusFail(prometheusService)
	return OnErrorFail(err, format, v...)
}

// OnARMAPISuccess ...
func OnARMAPISuccess(prometheusService string, format string, v ...interface{}) {
	PrometheusSuccess(prometheusService)
}

// NotFound ...
func NotFound(err error) bool {
	isDetailedError, _, detailedError := RetrieveRequestID(err)
	return isDetailedError && detailedError.Response.StatusCode == 404
}

// RunInParallel executes multiple functions (which return an error) as go functions concurrently.
func RunInParallel(funcs []func() error) error {
	//
	// Execute multiple functions (which return an error) as go functions concurrently.
	//
	var wg sync.WaitGroup
	wg.Add(len(funcs))

	errors := make([]error, len(funcs))
	for i, funOuter := range funcs {
		go func(results []error, idx int, funInner func() error) {
			defer wg.Done()
			if funInner == nil {
				results[idx] = fmt.Errorf("Received nil function")
				return
			}
			err := funInner()
			results[idx] = err
		}(errors, i, funOuter)
	}

	wg.Wait()

	var trimmedErrorMessages []string
	for _, e := range errors {
		if e != nil {
			trimmedErrorMessages = append(trimmedErrorMessages, e.Error())
		}
	}
	if len(trimmedErrorMessages) > 0 {
		return fmt.Errorf(strings.Join(trimmedErrorMessages, "\n"))
	}
	return nil
}
