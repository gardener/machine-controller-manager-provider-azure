// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
)

var (
	// ITResourceTagKey is specifically used for integration test
	// primarily to avoid orphan collection of resources when the control cluster is
	// non-seed cluster
	ITResourceTagKey = "kubernetes.io-role-integration-test"

	// ITResourceTagValue is specifically used for integration test
	// primarily to avoid orphan collection of resources when the control cluster is
	// non-seed cluster
	ITResourceTagValue = "1"
)

// ResourcesTrackerImpl implements the Resource Tracker Interface from the Integration test suite
type ResourcesTrackerImpl struct {
	MachineClass  *v1alpha1.MachineClass
	ResourceGroup string
	SecretData    map[string][]byte
}

// InitializeResourcesTracker is the constructor of ResourceTrackerImpl
// create a cleanup function to delete the list of orphan resources.
// 1. get list of orphan resources.
// 2. Mark them for deletion and call cleanup.
// 3. Print the orphan resources which got error in deletion.
func (r *ResourcesTrackerImpl) InitializeResourcesTracker(machineClass *v1alpha1.MachineClass, secretData map[string][]byte, clusterName string) error {

	r.MachineClass = machineClass
	r.SecretData = secretData
	r.ResourceGroup = clusterName // because the supplied cluster name is same as resource group name

	accessFactory, connectConfig := getAccessFactoryAndConfig(r.SecretData)
	ctx := context.TODO()
	initialVMs, initialVolumes, initialMachines, initialNICs, err := r.probeResources(ctx, accessFactory, connectConfig)
	if err != nil {
		fmt.Printf("Error in initial probe of orphaned resources: %s", err.Error())
		return err
	}

	delErrOrphanedVms, delErrOrphanedVolumes, delErrOrphanedNICs := cleanUpOrphanedResources(ctx, accessFactory, connectConfig, r.ResourceGroup, initialVMs, initialVolumes, initialNICs)

	if delErrOrphanedVms != nil || delErrOrphanedVolumes != nil || initialMachines != nil || delErrOrphanedNICs != nil {
		fmt.Printf("Error in deleting the following Orphan Resources")
		err := fmt.Errorf("virtual machines: %v\ndisks: %v\nnics: %v\nmcm machines: %v", delErrOrphanedVms, delErrOrphanedVolumes, delErrOrphanedNICs, initialMachines)
		return err
	}
	return nil

}

// IsOrphanedResourcesAvailable checks whether there are any orphaned resources left.
// If yes, then prints them and returns true. If not, then returns false.
func (r *ResourcesTrackerImpl) IsOrphanedResourcesAvailable() bool {
	accessFactory, connectConfig := getAccessFactoryAndConfig(r.SecretData)
	ctx := context.TODO()
	afterTestExecutionVMs, afterTestExecutionAvailDisks, afterTestExecutionAvailmachines, afterTestExecutionNICs, err := r.probeResources(ctx, accessFactory, connectConfig)
	if err != nil {
		fmt.Printf("Error probing orphaned resources: %s", err.Error())
		return true
	}

	if afterTestExecutionVMs != nil || afterTestExecutionAvailDisks != nil || afterTestExecutionNICs != nil || afterTestExecutionAvailmachines != nil {
		fmt.Printf("Following resources are orphaned\n")
		fmt.Printf("Virtual Machines: %v\nVolumes: %v\nNICs: %v\nMCM Machines: %v\n", afterTestExecutionVMs, afterTestExecutionAvailDisks, afterTestExecutionNICs, afterTestExecutionAvailmachines)
		return true
	}
	return false
}

// probeResources will look for orphaned resources and returns
// those in the order
// orphanedInstances, orphanedVolumes, orphanedMachines, orphanedNICs
func (r *ResourcesTrackerImpl) probeResources(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig) ([]string, []string, []string, []string, error) {
	accessFactory, connectConfig := getAccessFactoryAndConfig(r.SecretData)

	VMs, err := getOrphanedVMs(ctx, accessFactory, connectConfig, r.ResourceGroup)
	if err != nil {
		return VMs, nil, nil, nil, err
	}

	availVols, err := getOrphanedDisks(ctx, accessFactory, connectConfig, r.ResourceGroup)
	if err != nil {
		return VMs, availVols, nil, nil, err
	}

	availMachines, err := getMachines(ctx, factory, r.MachineClass, r.SecretData)
	if err != nil {
		return VMs, availVols, availMachines, nil, err
	}

	availNICs, err := getOrphanedNICs(ctx, factory, connectConfig, r.ResourceGroup)

	return VMs, availVols, availMachines, availNICs, err

}
