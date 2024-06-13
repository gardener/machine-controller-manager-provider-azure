// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	accesshelpers "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	listVmsNICsAndDisksQueryTemplate = `
	Resources
	| where type =~ 'microsoft.compute/virtualmachines' or type =~ 'microsoft.network/networkinterfaces' or type =~ 'microsoft.compute/disks'
	| where resourceGroup =~ '%s'
	| extend tagKeys = bag_keys(tags)
	| where tagKeys has '%s' and tagKeys has '%s'
	| project type, name
	`
)

// ExtractVMNamesFromVMsNICsDisks leverages resource graph to extract names from VMs, NICs and Disks (OS and Data disks).
func ExtractVMNamesFromVMsNICsDisks(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup string, providerSpec api.AzureProviderSpec) ([]string, error) {
	rgAccess, err := factory.GetResourceGraphAccess(connectConfig)
	if err != nil {
		return nil, err
	}
	vmNames := sets.New[string]()

	queryTemplateArgs := prepareQueryTemplateArgs(resourceGroup, providerSpec.Tags)
	resultEntries, err := accesshelpers.QueryAndMap[resultEntry](ctx, rgAccess, connectConfig.SubscriptionID, createVMNameMapperFn(), listVmsNICsAndDisksQueryTemplate, queryTemplateArgs...)
	if err != nil {
		return nil, status.WrapError(codes.Internal, fmt.Sprintf("failed to get VM names from VMs, NICs and Disks for resourceGroup :%s: error: %v", resourceGroup, err), err)
	}

	if resultEntries != nil {
		dataDiskNameSuffixes := getDataDiskNameSuffixes(providerSpec)
		for _, re := range resultEntries {
			vmName := re.extractVMName(dataDiskNameSuffixes)
			if !utils.IsEmptyString(vmName) {
				vmNames.Insert(vmName)
			}
		}
	}
	return vmNames.UnsortedList(), nil
}

func prepareQueryTemplateArgs(resourceGroup string, providerSpecTags map[string]string) []any {
	// NOTE: length is 3 because in the query we have a max of 3 parameter substitutions. This should be changed if the number of parameters change to prevent unnecessary resizing.
	templateArgs := make([]any, 0, 3)
	// NOTE: preserve the same order as these are ordered parameters which will be used for substitution.
	templateArgs = append(templateArgs, resourceGroup)
	for k := range providerSpecTags {
		if strings.HasPrefix(k, utils.ClusterTagPrefix) || strings.HasPrefix(k, utils.RoleTagPrefix) {
			templateArgs = append(templateArgs, k)
		}
	}
	return templateArgs
}

func createVMNameMapperFn() accesshelpers.MapperFn[resultEntry] {
	return func(m map[string]interface{}) *resultEntry {
		resourceName, nameKeyFound := m["name"].(string)
		resourceType, typeKeyFound := m["type"].(string)
		if nameKeyFound && typeKeyFound {
			return to.Ptr(resultEntry{
				resourceType: utils.ResourceType(resourceType),
				name:         resourceName,
			})
		}
		return nil
	}
}

type resultEntry struct {
	resourceType utils.ResourceType
	name         string
}

func (r resultEntry) extractVMName(dataDiskNameSuffixes sets.Set[string]) string {
	switch r.resourceType {
	case utils.VirtualMachinesResourceType:
		return r.name
	case utils.NetworkInterfacesResourceType:
		return utils.ExtractVMNameFromNICName(r.name)
	case utils.DiskResourceType:
		if strings.HasSuffix(r.name, utils.OSDiskSuffix) {
			return utils.ExtractVMNameFromOSDiskName(r.name)
		} else if strings.HasSuffix(r.name, utils.DataDiskSuffix) {
			if suffix, found := findMatchingDataDiskNameSuffix(r.name, dataDiskNameSuffixes); found {
				return r.name[:len(r.name)-len(suffix)]
			}
		}
	}
	return ""
}

func findMatchingDataDiskNameSuffix(dataDiskName string, dataDiskNameSuffixes sets.Set[string]) (string, bool) {
	for _, suffix := range dataDiskNameSuffixes.UnsortedList() {
		if strings.HasSuffix(dataDiskName, suffix) {
			return suffix, true
		}
	}
	return "", false
}

func getDataDiskNameSuffixes(providerSpec api.AzureProviderSpec) sets.Set[string] {
	dataDiskNameSuffixes := sets.New[string]()
	dataDisks := providerSpec.Properties.StorageProfile.DataDisks
	if dataDisks != nil {
		for _, dataDisk := range dataDisks {
			dataDiskNameSuffixes.Insert(utils.GetDataDiskNameSuffix(dataDisk.Name, dataDisk.Lun))
		}
	}
	return dataDiskNameSuffixes
}
