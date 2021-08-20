package provider

import (
	"fmt"

	provider "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

var (
	// cluster tag and cluster values are specifically used for integration test
	// in case the test is run against non seed cluster then the supplied MachineClass
	// is expected to have these tags set so that the machines from this suite won't be
	// orphan collected.
	clusterTag      = "mcm-integration-test"
	clusterTagValue = "true"
)

// ResourcesTrackerImpl implements the Resource Tracker Interface from the Integration test suite
type ResourcesTrackerImpl struct {
	ClusterName      string
	InitialVolumes   []string
	InitialInstances []string
	InitialMachines  []string
	MachineClass     *v1alpha1.MachineClass
	ResourceGroup    string
	SecretData       map[string][]byte
}

// InitializeResourceTracker is the constructor of ResourceTrackerImpl
func (r *ResourcesTrackerImpl) InitializeResourcesTracker(machineClass *v1alpha1.MachineClass, secretData map[string][]byte, clusterName string) error {

	r.MachineClass = machineClass
	r.SecretData = secretData
	r.ClusterName = clusterName

	providerSpec, err := provider.DecodeProviderSpecAndSecret(machineClass, &v1.Secret{Data: secretData})
	if err != nil {
		return err
	}
	r.ResourceGroup = providerSpec.ResourceGroup

	clients, err := getAzureClients(secretData)
	if err != nil {
		return err
	}

	instances, err := getVMsWithTag(clients, clusterTag, clusterTagValue, machineClass, r.ResourceGroup, secretData)
	if err == nil {
		r.InitialInstances = instances
		volumes, err := getAvailableDisks(clients, clusterTag, clusterTagValue, r.ResourceGroup)
		if err == nil {
			r.InitialVolumes = volumes
			r.InitialMachines, err = getMachines(machineClass, secretData)
			return err
		}
		return err
	}
	return err
}

// IsOrphanedResourcesAvailable checks whether there are any orphaned resources left.
// If yes, then prints them and returns true. If not, then returns false
func (r *ResourcesTrackerImpl) IsOrphanedResourcesAvailable() bool {
	afterTestExecutionInstances, afterTestExecutionAvailVols, afterTestExecutionAvailmachines, err := r.probeResources()
	//Check there is no error occured
	if err == nil {
		orphanedResourceInstances := differenceOrphanedResources(r.InitialInstances, afterTestExecutionInstances)
		if orphanedResourceInstances != nil {
			fmt.Println("orphaned instances were:", orphanedResourceInstances)
			return true
		}
		orphanedResourceAvailVols := differenceOrphanedResources(r.InitialVolumes, afterTestExecutionAvailVols)
		if orphanedResourceAvailVols != nil {
			fmt.Println("orphaned volumes were:", orphanedResourceAvailVols)
			return true
		}
		orphanedResourceAvailMachines := differenceOrphanedResources(r.InitialMachines, afterTestExecutionAvailmachines)
		if orphanedResourceAvailMachines != nil {
			fmt.Println("orphaned volumes were:", orphanedResourceAvailMachines)
			return true
		}
		return false
	}
	//assuming there are orphaned resources as probe can not be done
	return true
}

// probeResources will look for resources currently available and returns them
func (r *ResourcesTrackerImpl) probeResources() ([]string, []string, []string, error) {
	// Check for VM instances with matching tags/labels
	// Describe volumes attached to VM instance & delete the volumes
	// Finally delete the VM instance

	clients, err := getAzureClients(r.SecretData)
	if err != nil {
		return nil, nil, nil, err
	}

	instances, err := getVMsWithTag(clients, clusterTag, clusterTagValue, r.MachineClass, r.ResourceGroup, r.SecretData)
	if err != nil {
		return instances, nil, nil, err
	}

	// Check for available volumes in cloud provider with tag/label
	availVols, err := getAvailableDisks(clients, clusterTag, clusterTagValue, r.ResourceGroup)
	if err != nil {
		return instances, availVols, nil, err
	}

	// check for available machines
	availMachines, _ := getMachines(r.MachineClass, r.SecretData)

	// Check for available network interfaces in cloud provider with tag
	additionalResourcesCheck(clients, r.ResourceGroup, clusterTag, clusterTagValue)

	return instances, availVols, availMachines, err

}

// differenceOrphanedResources checks for difference in the found orphaned resource before test execution with the list after test execution
func differenceOrphanedResources(beforeTestExecution []string, afterTestExecution []string) []string {
	var diff []string

	// Loop two times, first to find beforeTestExecution strings not in afterTestExecution,
	// second loop to find afterTestExecution strings not in beforeTestExecution
	for i := 0; i < 2; i++ {
		for _, b1 := range beforeTestExecution {
			found := false
			for _, a2 := range afterTestExecution {
				if b1 == a2 {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, b1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			beforeTestExecution, afterTestExecution = afterTestExecution, beforeTestExecution
		}
	}

	return diff
}
