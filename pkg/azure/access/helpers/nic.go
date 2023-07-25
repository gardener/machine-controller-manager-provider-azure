package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

// labels used for recording prometheus metrics
const (
	subnetGetServiceLabel = "subnet_get"
	nicGetServiceLabel    = "nic_get"
	nicDeleteServiceLabel = "nic_delete"
	nicCreateServiceLabel = "nic_create"
)

const (
	defaultDeleteNICTimeout = 10 * time.Minute
	defaultCreateNICTimeout = 15 * time.Minute
)

func DeleteNIC(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroup, nicName string) (err error) {
	defer instrument.RecordAzAPIMetric(err, nicDeleteServiceLabel, time.Now())
	var poller *runtime.Poller[armnetwork.InterfacesClientDeleteResponse]
	delCtx, cancelFn := context.WithTimeout(ctx, defaultDeleteNICTimeout)
	defer cancelFn()
	poller, err = client.BeginDelete(delCtx, resourceGroup, nicName, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger delete of NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
		return
	}
	_, err = poller.PollUntilDone(delCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Deleting of NIC: %s for ResourceGroup: %s", nicName, resourceGroup)
	}
	return
}

//// CreateNICIfNotExists creates a NIC if it does not exist. It returns the NIC ID of an already existing NIC or a freshly created one.
//func CreateNICIfNotExists(ctx context.Context, nicAccess *armnetwork.InterfacesClient, subnet *armnetwork.Subnet, providerSpec api.AzureProviderSpec, nicName string) (string, error) {
//	resourceGroup := providerSpec.ResourceGroup
//	nic, err := getNIC(ctx, nicAccess, resourceGroup, nicName)
//	if err != nil {
//		return "", err
//	}
//	if nic != nil {
//		return *nic.ID, nil
//	}
//	// NIC is not found, create NIC
//	nicParams := createNICParams(providerSpec, subnet, nicName)
//	nic, err = createNIC(ctx, nicAccess, resourceGroup, nicParams)
//	if err != nil {
//		return "", err
//	}
//	return *nic.ID, nil
//}

func GetNIC(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroup, nicName string) (nic *armnetwork.Interface, err error) {
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

func CreateNIC(ctx context.Context, nicAccess *armnetwork.InterfacesClient, providerSpec api.AzureProviderSpec, subnet *armnetwork.Subnet, nicName string) (nic *armnetwork.Interface, err error) {
	defer instrument.RecordAzAPIMetric(err, nicCreateServiceLabel, time.Now())
	var (
		poller       *runtime.Poller[armnetwork.InterfacesClientCreateOrUpdateResponse]
		creationResp armnetwork.InterfacesClientCreateOrUpdateResponse
	)
	createCtx, cancelFn := context.WithTimeout(ctx, defaultCreateNICTimeout)
	defer cancelFn()

	nicParams := createNICParams(providerSpec, subnet, nicName)
	resourceGroup := providerSpec.ResourceGroup

	poller, err = nicAccess.BeginCreateOrUpdate(createCtx, resourceGroup, nicName, nicParams, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to trigger create of NIC [ResourceGroup: %s, Name: %s]", resourceGroup, nicName)
		return nil, err
	}
	creationResp, err = poller.PollUntilDone(createCtx, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Polling failed while waiting for Creation of NIC: %s for ResourceGroup: %s", nicName, resourceGroup)
	}
	nic = &creationResp.Interface
	return
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
