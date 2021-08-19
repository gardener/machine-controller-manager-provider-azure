package provider

import (
	"fmt"

	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
)

var (
	// ITResourceTagKey and ITResourceTagValue are specifically used for integration test
	// primarily to avoid orphan collection of resources when the control cluster is
	// non seed cluster
	ITResourceTagKey   = "kubernetes.io-role-integration-test"
	ITResourceTagValue = "1"
)

// ResourcesTrackerImpl implements the Resource Tracker Interface from the Integration test suite
type ResourcesTrackerImpl struct {
	MachineClass  *v1alpha1.MachineClass
	ResourceGroup string
	SecretData    map[string][]byte
}

// InitializeResourceTracker is the constructor of ResourceTrackerImpl
func (r *ResourcesTrackerImpl) InitializeResourcesTracker(machineClass *v1alpha1.MachineClass, secretData map[string][]byte, clusterName string) error {

	r.MachineClass = machineClass
	r.SecretData = secretData
	r.ResourceGroup = clusterName // because the supplied cluster name is same as resource group name

	initialVMs, initialVolumes, initialMachines, initialNICs, err := r.probeResources()
	if err != nil {
		fmt.Printf("Error in initial probe of orphaned resources: %s", err.Error())
		return err
	}

	if initialVMs != nil || initialVolumes != nil || initialMachines != nil || initialNICs != nil {
		fmt.Printf("Orphan resources are available. Clean them up before proceeding with the test.")
		err := fmt.Errorf("virtual machines: %v\ndisks: %v\nnics: %v\nmcm machines: %v", initialVMs, initialVolumes, initialNICs, initialMachines)
		return err
	}
	return nil

}

// IsOrphanedResourcesAvailable checks whether there are any orphaned resources left.
// If yes, then prints them and returns true. If not, then returns false.
func (r *ResourcesTrackerImpl) IsOrphanedResourcesAvailable() bool {
	afterTestExecutionVMs, afterTestExecutionAvailDisks, afterTestExecutionAvailmachines, afterTestExecutionNICs, err := r.probeResources()
	if err != nil {
		fmt.Printf("Error probing orphaned resources: %s", err.Error())
		return true
	}

	if afterTestExecutionVMs != nil || afterTestExecutionAvailDisks != nil || afterTestExecutionNICs != nil || afterTestExecutionAvailmachines != nil {
		fmt.Printf("attempting to delete the following resources are orphaned\n")
		fmt.Printf("Virtual Machines: %v\nVolumes: %v\nNICs: %v\nMCM Machines: %v\n", afterTestExecutionVMs, afterTestExecutionAvailDisks, afterTestExecutionNICs, afterTestExecutionAvailmachines)
		return true
	}
	return false
}

// probeResources will look for orphaned resources and returns
// those resources which could not be deleted in the order
// orphanedInstances, orphanedVolumes, orphanedMachines, orphanedNICs
func (r *ResourcesTrackerImpl) probeResources() ([]string, []string, []string, []string, error) {

	clients, err := getAzureClients(r.SecretData)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	VMs, err := getOrphanedVMs(clients, ITResourceTagKey, ITResourceTagValue, r.MachineClass, r.ResourceGroup, r.SecretData)
	if err != nil {
		return VMs, nil, nil, nil, err
	}

	availVols, err := getOrphanedDisks(clients, ITResourceTagKey, ITResourceTagValue, r.ResourceGroup)
	if err != nil {
		return VMs, availVols, nil, nil, err
	}

	availMachines, err := getMachines(r.MachineClass, r.SecretData)
	if err != nil {
		return VMs, availVols, availMachines, nil, err
	}

	availNICs, err := getOrphanedNICs(clients, r.ResourceGroup, ITResourceTagKey, ITResourceTagValue)

	return VMs, availVols, availMachines, availNICs, err

}
