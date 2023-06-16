package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

const (
	listVMsQueryTemplate = `
	Resources
	| where type =~ 'Microsoft.Compute/virtualMachines'
	| where resourceGroup =~ '%s'
	| extend tagKeys = bag_keys(tags)
	| where tagKeys hasprefix "kubernetes.io-cluster-" and tagKeys hasprefix "kubernetes.io-role-"
	| project name
	`
	listNICsQueryTemplate = `
	Resources
	| where type =~ 'microsoft.network/networkinterfaces'
	| where resourceGroup =~ '%s'
	| extend tagKeys = bag_keys(tags)
	| where tagKeys hasprefix "kubernetes.io-cluster-" and tagKeys hasprefix "kubernetes.io-role-"
	| project name
	`
	nicSuffix = "-nic"
)

// vmNameExtractorFn is a function which takes a name of a resource and extracts a VM name from it.
type vmNameExtractorFn func(string) (string, bool)

// ExtractVMNamesFromVirtualMachinesAndNICs extracts VM names from virtual machines and NIC names and returns a slice of unique vm names.
func ExtractVMNamesFromVirtualMachinesAndNICs(ctx context.Context, client *armresourcegraph.Client, subscriptionID, resourceGroup string) ([]string, error) {
	vmNames := sets.New[string]()
	vmNamesFromVirtualMachines, err := doExtractVMNamesFromResource(ctx, client, subscriptionID, resourceGroup, listVMsQueryTemplate, nil)
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
	// and have a slice of such MachineInfo returned as part of this processor method.
	vmNamesFromNICs, err := doExtractVMNamesFromResource(ctx, client, subscriptionID, resourceGroup, listNICsQueryTemplate, vmNameExtractorFromNIC)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get VM names from NICs for resourceGroup :%s: error: %v", resourceGroup, err))
	}
	vmNames.Insert(vmNamesFromNICs...)
	return vmNames.UnsortedList(), nil
}

//type GraphQueryExecutor[T any] struct {
//	Client         *armresourcegraph.Client
//	SubscriptionID string
//}
//
//
//func (g *GraphQueryExecutor[T]) Execute(ctx context.Context, query string, mapperFn MapperFn[T]) T {
//
//}

type MapperFn[T any] func(map[string]interface{}) T

func QueryAndMap[T any](ctx context.Context, client *armresourcegraph.Client, subscriptionID, query string, mapperFn MapperFn[T]) *T {
	return nil
}

type ResourceGraphQueryExecutor interface {
	Execute(ctx context.Context, subscriptionID, query string) ([]interface{}, error)
}

// doExtractVMNamesFromResource queries for resources using the given queryTemplate and extracts VM names from the list of resources retrieved.
func doExtractVMNamesFromResource(ctx context.Context, client *armresourcegraph.Client, subscriptionID, resourceGroup, queryTemplate string, extractorFn vmNameExtractorFn) ([]string, error) {
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
	var resourceNames []string
	if resources.TotalRecords == pointer.Int64(0) {
		return resourceNames, nil
	}

	// resourceResponse.Data is a []interface{}
	if objSlice, ok := resources.Data.([]interface{}); ok {
		for _, obj := range objSlice {
			// Each obj in resourceResponse.Data is a map[string]Interface{}
			rowElements := obj.(map[string]interface{})
			if resourceNameVal, keyFound := rowElements["name"]; keyFound {
				resourceName := resourceNameVal.(string)
				if extractorFn != nil {
					if extractedName, extracted := extractorFn(resourceName); extracted {
						resourceNames = append(resourceNames, extractedName)
					}
				} else {
					resourceNames = append(resourceNames, resourceName)
				}
			}
		}
	}
	return resourceNames, nil
}

// vmNameExtractorFromNIC extracts VM name from NIC name.
func vmNameExtractorFromNIC(nicName string) (string, bool) {
	if strings.HasSuffix(nicName, nicSuffix) {
		return nicName[:len(nicName)-len(nicSuffix)], true
	}
	return "", false
}
