package utils

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

const (
	// ideally we should not be using tags whose keys have fixed prefix but a dynamic suffix. Keys should always be fixed and their values should be dynamic.
	// due to this we now have to complicate the KUSTO query by using `mv-expand` which gives access to keys and values for tags separately and then one can use `startswith` to apply the tag filters.
	listVMsQueryTemplate = `
	Resources
	| where type =~ 'Microsoft.Compute/virtualMachines'
	| where resourceGroup =~ '%s'
	| where bag_keys(tags) hasprefix "kubernetes.io-cluster-"
	| where bag_keys(tags) hasprefix "kubernetes.io-role-"
	| project name
	`
	listNICsQueryTemplate = `
	Resources
	| where type =~ 'microsoft.network/networkinterfaces'
	| where resourceGroup =~ '%s'
	| where bag_keys(tags) hasprefix "kubernetes.io-cluster-"
	| where bag_keys(tags) hasprefix "kubernetes.io-role-"
	| project name
	`
)

func ExtractVMNamesFromVirtualMachinesAndNICs(ctx context.Context, client *armresourcegraph.Client, subscriptionID, resourceGroup string) ([]string, error) {
	vmNames := sets.New[string]()
	vmNamesFromVirtualMachines, err := doExtractVMNamesFromResource(ctx, client, subscriptionID, resourceGroup, listVMsQueryTemplate)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get VM names from VirtualMachines for resourceGroup :%s: error: %v", resourceGroup, err))
	}
	vmNames.Insert(vmNamesFromVirtualMachines...)

	// extract VM Names from existing NICs. Why is this required?
	// A Machine in MCM terminology is a collective entity consisting of but not limited to VM, NIC(s), Disk(s).
	// MCM orphan collection needs to track resources which have a separate lifecycle (currently in case of azure it is VM's and NICs.
	// Disks (OS and Data) are created and deleted along with then VM.) and which are now orphaned. Unfortunately, MCM only orphan collects
	// machines (a collective resource) and a machine is uniquely identified by a VM name (again not so ideal).
	// In order to get any orphaned VM or NIC, its currently essential that a VM name which serves as a unique machine name should be collected
	// by introspecting VMs and NICs. Ideally you would change the response struct to separately capture VM name(s) and NIC name(s) under MachineInfo
	// and have a slice of such MachineInfo returned as part of this driver method.
	vmNamesFromNICs, err := doExtractVMNamesFromResource(ctx, client, subscriptionID, resourceGroup, listNICsQueryTemplate)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get VM names from NICs for resourceGroup :%s: error: %v", resourceGroup, err))
	}
	vmNames.Insert(vmNamesFromNICs...)
	return vmNames.UnsortedList(), nil
}

func doExtractVMNamesFromResource(ctx context.Context, client *armresourcegraph.Client, subscriptionID, resourceGroup, queryTemplate string) ([]string, error) {
	// azure resource graph uses KUSTO as their queryTemplate language.
	// For additional information on KUSTO start here: [https://learn.microsoft.com/en-us/azure/data-explorer/kusto/query/]
	resources, err := client.Resources(ctx,
		armresourcegraph.QueryRequest{
			Query:         to.Ptr(fmt.Sprintf(queryTemplate, resourceGroup)),
			Options:       nil,
			Subscriptions: []*string{to.Ptr(subscriptionID)},
		}, nil)

	if err != nil {
		return nil, err
	}
	var vmNames []string
	if resources.TotalRecords == pointer.Int64(0) {
		return vmNames, nil
	}

	// resourceResponse.Data is a []interface{}
	if objSlice, ok := resources.Data.([]interface{}); ok {
		for _, obj := range objSlice {
			// Each obj in resourceResponse.Data is a map[string]Interface{}
			rowElements := obj.(map[string]interface{})
			if vmNameVal, keyFound := rowElements["name"]; keyFound {
				vmName := vmNameVal.(string)
				vmNames = append(vmNames, vmName)
			}
		}
	}
	return vmNames, nil
}
