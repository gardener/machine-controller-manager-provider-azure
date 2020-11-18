package mock

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	clientutils "github.com/gardener/machine-controller-manager-provider-azure/pkg/client"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog"
)

// ListMachines lists the machines possibly created by a providerSpec
func (ms *PluginSPIImpl) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("List machines request has been recieved for %q", req.MachineClass.Name)
	defer klog.V(2).Infof("List machines request has been recieved for %q", req.MachineClass.Name)

	providerSpec, err := decodeProviderSpecAndSecret(req.MachineClass, req.Secret)
	ms.AzureProviderSpec = providerSpec

	var (
		resourceGroupName = providerSpec.ResourceGroup
		items             []compute.VirtualMachine
		listOfVMs         = make(map[string]string)
	)

	clients, err := ms.SPI.Setup(req.Secret)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	items, err = clients.VM.List(ctx, resourceGroupName)

	for _, item := range items {
		listOfVMs[encodeMachineID(*item.Location, *item.Name)] = *item.Name
	}

	clientutils.OnARMAPISuccess(prometheusServiceVM, "VM.List")
	return &driver.ListMachinesResponse{MachineList: listOfVMs}, nil
}

// GetMachineStatus handles a machine get status request
func (ms *PluginSPIImpl) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	var machineStatusResponse = &driver.GetMachineStatusResponse{}

	listMachineRequest := &driver.ListMachinesRequest{MachineClass: req.MachineClass, Secret: req.Secret}

	machines, err := ms.ListMachines(ctx, listMachineRequest)
	if err != nil {
		return nil, err
	}
	for providerID, VMName := range machines.MachineList {
		if VMName == req.Machine.Name {
			machineStatusResponse.NodeName = VMName
			machineStatusResponse.ProviderID = providerID
			return machineStatusResponse, nil
		}
	}

	err = fmt.Errorf("Machine '%s' not found", req.Machine.Name)
	return nil, status.Error(codes.NotFound, err.Error())
}
