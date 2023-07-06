package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	subnetGetServiceLabel = "subnet_get"
	nicGetServiceLabel    = "nic_get"
	nicDeleteServiceLabel = "nic_delete"
	nicCreateServiceLabel = "nic_create"
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

// CreateNICIfNotExists creates a NIC if it does not exist. It returns the NIC ID of an already existing NIC or a freshly created one.
func CreateNICIfNotExists(ctx context.Context, nicAccess *armnetwork.InterfacesClient, subnetAccess *armnetwork.SubnetsClient, providerSpec api.AzureProviderSpec, nicName string) (string, error) {
	resourceGroup := providerSpec.ResourceGroup
	nic, err := getNIC(ctx, nicAccess, resourceGroup, nicName)
	if err != nil {
		return "", err
	}
	if nic != nil {
		return *nic.ID, nil
	}
	// NIC is not found, create NIC
	subnetInfo := providerSpec.SubnetInfo
	subnetResourceGroup := getSubnetResourceGroup(resourceGroup, subnetInfo)
	subnet, err := getSubnet(ctx, subnetAccess, subnetResourceGroup, subnetInfo.VnetName, subnetInfo.SubnetName)
	if err != nil {
		return "", err
	}
	nicParams := createNICParams(providerSpec, subnet, nicName)
	nic, err = createNIC(ctx, nicAccess, resourceGroup, nicParams)
	if err != nil {
		return "", err
	}
	return *nic.ID, nil
}

func createNICParams(providerSpec api.AzureProviderSpec, subnet *armnetwork.Subnet, nicName string) armnetwork.Interface {
	return armnetwork.Interface{
		Location: to.Ptr(providerSpec.Location),
		Properties: &armnetwork.InterfacePropertiesFormat{
			EnableAcceleratedNetworking: providerSpec.Properties.NetworkProfile.AcceleratedNetworking,
			EnableIPForwarding:          to.Ptr(true),
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: &nicName,
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
						Subnet:                    subnet,
					},
				},
			},
			NicType: to.Ptr(armnetwork.NetworkInterfaceNicTypeStandard),
		},
		Tags: createNICTags(providerSpec.Tags),
		Name: &nicName,
	}
}

func createNICTags(tags map[string]string) map[string]*string {
	nicTags := make(map[string]*string, len(tags))
	for k, v := range tags {
		nicTags[k] = to.Ptr(v)
	}
	return nicTags
}

// getSubnetResourceGroup gets the resource group for the subnet.
// It is possible that a machine is assigned to a vnet in a different resource group. If a resource group has been
// set in api.AzureSubnetInfo then that is preferred.
func getSubnetResourceGroup(resourceGroup string, subnetInfo api.AzureSubnetInfo) string {
	rg := resourceGroup
	if subnetInfo.VnetResourceGroup != nil {
		rg = *subnetInfo.VnetResourceGroup
	}
	return rg
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

func createNIC(ctx context.Context, nicAccess *armnetwork.InterfacesClient, resourceGroup string, nicParams armnetwork.Interface) (nic *armnetwork.Interface, err error) {
	defer instrument.RecordAzAPIMetric(err, nicCreateServiceLabel, time.Now())
	var (
		poller       *runtime.Poller[armnetwork.InterfacesClientCreateOrUpdateResponse]
		creationResp armnetwork.InterfacesClientCreateOrUpdateResponse
	)
	nicName := *nicParams.Name
	poller, err = nicAccess.BeginCreateOrUpdate(ctx, resourceGroup, nicName, nicParams, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger create of NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
		return nil, err
	}
	creationResp, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Creation of NIC: %s for ResourceGroup: %s", nicName, resourceGroup)
	}
	nic = &creationResp.Interface
	return
}

func getSubnet(ctx context.Context, subnetAccess *armnetwork.SubnetsClient, resourceGroup, virtualNetworkName, subnetName string) (subnet *armnetwork.Subnet, err error) {
	var subnetResp armnetwork.SubnetsClientGetResponse
	defer instrument.RecordAzAPIMetric(err, subnetGetServiceLabel, time.Now())
	subnetResp, err = subnetAccess.Get(ctx, resourceGroup, virtualNetworkName, subnetName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to GET Subnet for [resourceGroup: %s, virtualNetworkName: %s, subnetName: %s]", resourceGroup, virtualNetworkName, subnetName)
		return nil, err
	}
	subnet = &subnetResp.Subnet
	return
}
