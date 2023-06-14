package azure

import (
	"context"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/types"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
)

// reqProcessor implements driver.Driver interface
type reqProcessor struct {
	clientProvider types.ClientProvider
}

// NewRequestProcessor creates a new instance of an implementation of driver.Driver
func NewRequestProcessor(clientProvider types.ClientProvider) driver.Driver {
	return reqProcessor{clientProvider: clientProvider}
}

func (d reqProcessor) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	providerSpec, connectConfig, err := utils.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}
	// azure resource graph uses KUSTO as their query language.
	// For additional information on KUSTO start here: [https://learn.microsoft.com/en-us/azure/data-explorer/kusto/query/]
	client, err := d.clientProvider.CreateResourceGraphClient(*connectConfig)
	if err != nil {
		return nil, err
	}
	vmNames, err := utils.ExtractVMNamesFromVirtualMachinesAndNICs(ctx, client, connectConfig.SubscriptionID, providerSpec.ResourceGroup)
	if err != nil {
		return nil, err
	}
	return utils.CreateMachineListResponse(providerSpec.Location, vmNames)
}

func (d reqProcessor) CreateMachine(ctx context.Context, request *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d reqProcessor) DeleteMachine(ctx context.Context, request *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d reqProcessor) GetMachineStatus(ctx context.Context, request *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d reqProcessor) GetVolumeIDs(ctx context.Context, request *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
	//TODO implement me
	panic("implement me")
}
