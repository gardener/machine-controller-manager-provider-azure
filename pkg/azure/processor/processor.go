package processor

import (
	"context"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client"
	clienthelpers "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/processor/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
)

// reqProcessor implements processor.Driver interface
type reqProcessor struct {
	clientProvider client.ClientProvider
}

// NewRequestProcessor creates a new instance of an implementation of processor.Driver
func NewRequestProcessor(clientProvider client.ClientProvider) driver.Driver {
	return reqProcessor{clientProvider: clientProvider}
}

func (d reqProcessor) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}
	// azure resource graph uses KUSTO as their query language.
	// For additional information on KUSTO start here: [https://learn.microsoft.com/en-us/azure/data-explorer/kusto/query/]
	client, err := d.clientProvider.CreateResourceGraphClient(*connectConfig)
	if err != nil {
		return nil, err
	}
	vmNames, err := clienthelpers.ExtractVMNamesFromVirtualMachinesAndNICs(ctx, client, connectConfig.SubscriptionID, providerSpec.ResourceGroup)
	if err != nil {
		return nil, err
	}
	return helpers.CreateMachineListResponse(providerSpec.Location, vmNames)
}

func (d reqProcessor) CreateMachine(ctx context.Context, request *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d reqProcessor) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	//providerSpec, connectConfig, err := helpers.ExtractProviderSpecAndConnectConfig(req.MachineClass, req.Secret)
	//if err != nil {
	//	return nil, err
	//}

	//TODO implement me
	panic("implement me")
}

func (d reqProcessor) GetMachineStatus(ctx context.Context, request *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d reqProcessor) GetVolumeIDs(_ context.Context, request *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
	const csiDriverName = "disk.csi.azure.com"
	var volumeIDs []string

	if request.PVSpecs != nil {
		for _, pvSpec := range request.PVSpecs {
			if pvSpec.AzureDisk != nil {
				volumeIDs = append(volumeIDs, pvSpec.AzureDisk.DiskName)
			} else if pvSpec.CSI != nil && pvSpec.CSI.Driver == csiDriverName && !utils.IsEmptyString(pvSpec.CSI.VolumeHandle) {
				volumeIDs = append(volumeIDs, pvSpec.CSI.VolumeHandle)
			}
		}
	}

	return &driver.GetVolumeIDsResponse{VolumeIDs: volumeIDs}, nil
}
