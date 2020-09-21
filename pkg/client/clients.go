package client

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/marketplaceordering/mgmt/marketplaceordering"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
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

// AzureDriverClients . . .
type AzureDriverClients struct {
	Subnet      network.SubnetsClient
	Nic         network.InterfacesClient
	VM          compute.VirtualMachinesClient
	Disk        compute.DisksClient
	Deployments resources.DeploymentsClient
	Images      compute.VirtualMachineImagesClient
	Marketplace marketplaceordering.MarketplaceAgreementsClient
}

func PrometheusFail(service string) {
	metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "azure", "service": service}).Inc()
}

func PrometheusSuccess(service string) {
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "azure", "service": service}).Inc()
}

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

func OnARMAPIErrorFail(prometheusService string, err error, format string, v ...interface{}) error {
	PrometheusFail(prometheusService)
	return OnErrorFail(err, format, v...)
}

func OnARMAPISuccess(prometheusService string, format string, v ...interface{}) {
	PrometheusSuccess(prometheusService)
}

func (clients *AzureDriverClients) waitForDataDiskDetachment(ctx context.Context, resourceGroupName string, vm compute.VirtualMachine) error {
	klog.V(2).Infof("Data disk detachment began for %q", *vm.Name)
	defer klog.V(2).Infof("Data disk detached for %q", *vm.Name)

	if len(*vm.StorageProfile.DataDisks) > 0 {
		// There are disks attached hence need to detach them
		vm.StorageProfile.DataDisks = &[]compute.DataDisk{}

		future, err := clients.VM.CreateOrUpdate(ctx, resourceGroupName, *vm.Name, vm)
		if err != nil {
			return OnARMAPIErrorFail(prometheusServiceVM, err, "Failed to CreateOrUpdate. Error Message - %s", err)
		}
		err = future.WaitForCompletionRef(ctx, clients.VM.Client)
		if err != nil {
			return OnARMAPIErrorFail(prometheusServiceVM, err, "Failed to CreateOrUpdate. Error Message - %s", err)
		}
		OnARMAPISuccess(prometheusServiceVM, "VM CreateOrUpdate was successful for %s", *vm.Name)
	}

	return nil
}

func NotFound(err error) bool {
	isDetailedError, _, detailedError := RetrieveRequestID(err)
	return isDetailedError && detailedError.Response.StatusCode == 404
}

func (clients *AzureDriverClients) deleteVM(ctx context.Context, resourceGroupName string, vmName string) error {
	klog.V(2).Infof("VM deletion has began for %q", vmName)
	defer klog.V(2).Infof("VM deleted for %q", vmName)

	future, err := clients.VM.Delete(ctx, resourceGroupName, vmName)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceVM, err, "vm.Delete")
	}
	err = future.WaitForCompletionRef(ctx, clients.VM.Client)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceVM, err, "vm.Delete")
	}
	OnARMAPISuccess(prometheusServiceVM, "VM deletion was successful for %s", vmName)
	return nil
}

func (clients *AzureDriverClients) fetchAttachedVMfromNIC(ctx context.Context, resourceGroupName, nicName string) (string, error) {
	nic, err := clients.Nic.Get(ctx, resourceGroupName, nicName, "")
	if err != nil {
		return "", err
	}
	if nic.VirtualMachine == nil {
		return "", nil
	}
	return *nic.VirtualMachine.ID, nil
}

func (clients *AzureDriverClients) deleteNIC(ctx context.Context, resourceGroupName string, nicName string) error {
	klog.V(2).Infof("NIC delete started for %q", nicName)
	defer klog.V(2).Infof("NIC deleted for %q", nicName)

	future, err := clients.Nic.Delete(ctx, resourceGroupName, nicName)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.Delete")
	}
	if err := future.WaitForCompletionRef(ctx, clients.Nic.Client); err != nil {
		return OnARMAPIErrorFail(prometheusServiceNIC, err, "nic.Delete")
	}
	OnARMAPISuccess(prometheusServiceNIC, "NIC deletion was successful for %s", nicName)
	return nil
}

func (clients *AzureDriverClients) fetchAttachedVMfromDisk(ctx context.Context, resourceGroupName, diskName string) (string, error) {
	disk, err := clients.Disk.Get(ctx, resourceGroupName, diskName)
	if err != nil {
		return "", err
	}
	if disk.ManagedBy == nil {
		return "", nil
	}
	return *disk.ManagedBy, nil
}

func (clients *AzureDriverClients) deleteDisk(ctx context.Context, resourceGroupName string, diskName string) error {
	klog.V(2).Infof("Disk delete started for %q", diskName)
	defer klog.V(2).Infof("Disk deleted for %q", diskName)

	future, err := clients.Disk.Delete(ctx, resourceGroupName, diskName)
	if err != nil {
		return OnARMAPIErrorFail(prometheusServiceDisk, err, "disk.Delete")
	}
	if err = future.WaitForCompletionRef(ctx, clients.Disk.Client); err != nil {
		return OnARMAPIErrorFail(prometheusServiceDisk, err, "disk.Delete")
	}
	OnARMAPISuccess(prometheusServiceDisk, "Disk deletion was successful for %s", diskName)
	return nil
}

func (clients *AzureDriverClients) getDeleterForDisk(ctx context.Context, resourceGroupName string, diskName string) func() error {
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

func runInParallel(funcs []func() error) error {
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

// DeleteVMNicDisks deletes the VM and associated Disks and NIC
func (clients *AzureDriverClients) DeleteVMNicDisks(ctx context.Context, resourceGroupName string, VMName string, nicName string, diskName string, dataDiskNames []string) error {

	// We try to fetch the VM, detach its data disks and finally delete it
	if vm, vmErr := clients.VM.Get(ctx, resourceGroupName, VMName, ""); vmErr == nil {

		clients.waitForDataDiskDetachment(ctx, resourceGroupName, vm)
		if deleteErr := clients.deleteVM(ctx, resourceGroupName, VMName); deleteErr != nil {
			return deleteErr
		}

		OnARMAPISuccess(prometheusServiceVM, "VM Get was successful for %s", *vm.Name)
	} else if !NotFound(vmErr) {
		// If some other error occurred, which is not 404 Not Found (the VM doesn't exist) then bubble up
		return OnARMAPIErrorFail(prometheusServiceVM, vmErr, "vm.Get")
	}

	// Fetch the NIC and deleted it
	nicDeleter := func() error {
		if vmHoldingNic, err := clients.fetchAttachedVMfromNIC(ctx, resourceGroupName, nicName); err != nil {
			if NotFound(err) {
				// Resource doesn't exist, no need to delete
				return nil
			}
			return err
		} else if vmHoldingNic != "" {
			return fmt.Errorf("Cannot delete NIC %s because it is attached to VM %s", nicName, vmHoldingNic)
		}

		return clients.deleteNIC(ctx, resourceGroupName, nicName)
	}

	// Fetch the system disk and delete it
	diskDeleter := clients.getDeleterForDisk(ctx, resourceGroupName, diskName)

	deleters := []func() error{nicDeleter, diskDeleter}

	if dataDiskNames != nil {
		for _, dataDiskName := range dataDiskNames {
			dataDiskDeleter := clients.getDeleterForDisk(ctx, resourceGroupName, dataDiskName)
			deleters = append(deleters, dataDiskDeleter)
		}
	}

	return runInParallel(deleters)
}
