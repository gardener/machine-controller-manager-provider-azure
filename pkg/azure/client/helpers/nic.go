package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/client/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	subnetGetServiceLabel = "subnet_get"
	nicGetServiceLabel    = "nic_get"
	nicDeleteServiceLabel = "nic_delete"
)

func DeleteNICIfExists(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroup, nicName string) error {
	nic, err := getNIC(ctx, client, resourceGroup, nicName)
	if err != nil {
		return err
	}
	if nic == nil {
		klog.Infof("NIC: [ResourceGraph: %s, NICName: %s] does not exist. Skipping deletion.", resourceGroup, nicName)
		return nil
	}
	if nic.Properties != nil && nic.Properties.VirtualMachine != nil && nic.Properties.VirtualMachine.ID != nil {
		return fmt.Errorf("cannot delete NIC [ResourceGroup: %s, Name: %s] as its still attached to VM: %s", resourceGroup, nicName, *nic.Properties.VirtualMachine.ID)
	}
	err = deleteNIC(ctx, client, resourceGroup, nicName)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to delete NIC: [ResourceGroup: %s, NICName: %s] Err: %v", resourceGroup, nicName, err))
	}
	klog.Infof("Successfully delete NIC: [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
	return nil
}

func getNIC(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroup, nicName string) (nic *armnetwork.Interface, err error) {
	defer instrument.RecordAzAPIMetric(err, nicGetServiceLabel, time.Now())
	resp, err := client.Get(ctx, resourceGroup, nicName, nil)
	if err != nil {
		if errors.IsNotFoundAzAPIError(err) {
			return nil, nil
		}
		errors.LogAzAPIError(err, "Failed to get NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
		return nil, err
	}
	return &resp.Interface, nil
}

func deleteNIC(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroup, nicName string) (err error) {
	defer instrument.RecordAzAPIMetric(err, nicDeleteServiceLabel, time.Now())
	var poller *runtime.Poller[armnetwork.InterfacesClientDeleteResponse]
	poller, err = client.BeginDelete(ctx, resourceGroup, nicName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger delete of NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
		return
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Deleting of NIC: %s for ResourceGroup: %s", nicName, resourceGroup)
	}
	return
}

func getSubnet(ctx context.Context, client *armnetwork.SubnetsClient, resourceGroup, virtualNetworkName, subnetName string) (subnet *armnetwork.Subnet, err error) {
	var subnetResp armnetwork.SubnetsClientGetResponse
	defer instrument.RecordAzAPIMetric(err, subnetGetServiceLabel, time.Now())
	subnetResp, err = client.Get(ctx, resourceGroup, virtualNetworkName, subnetName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to GET Subnet for [resourceGroup: %s, virtualNetworkName: %s, subnetName: %s]", resourceGroup, virtualNetworkName, subnetName)
		return nil, err
	}
	subnet = &subnetResp.Subnet
	return
}
