package helpers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"k8s.io/utils/pointer"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	accesshelpers "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/validation"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func ExtractProviderSpecAndConnectConfig(mcc *v1alpha1.MachineClass, secret *corev1.Secret) (api.AzureProviderSpec, access.ConnectConfig, error) {
	var (
		err           error
		providerSpec  api.AzureProviderSpec
		connectConfig access.ConnectConfig
	)
	// validate provider Spec provider. Exit early if it is not azure.
	if err = validation.ValidateMachineClassProvider(mcc); err != nil {
		return providerSpec, connectConfig, err
	}
	// unmarshall raw provider Spec from MachineClass and validate it. If validation fails return an error else return decoded spec.
	if providerSpec, err = DecodeAndValidateMachineClassProviderSpec(mcc); err != nil {
		return api.AzureProviderSpec{}, access.ConnectConfig{}, err
	}
	// validate secret and extract connect config required to create clients.
	if connectConfig, err = ValidateSecretAndCreateConnectConfig(secret); err != nil {
		return api.AzureProviderSpec{}, access.ConnectConfig{}, err
	}
	return providerSpec, connectConfig, nil
}

func ConstructMachineListResponse(location string, vmNames []string) *driver.ListMachinesResponse {
	listMachineRes := driver.ListMachinesResponse{}
	instanceIdToVMNameMap := make(map[string]string, len(vmNames))
	if len(vmNames) == 0 {
		return &listMachineRes
	}
	for _, vmName := range vmNames {
		instanceIdToVMNameMap[DeriveInstanceID(location, vmName)] = vmName
	}
	listMachineRes.MachineList = instanceIdToVMNameMap
	return &listMachineRes
}

func ConstructGetMachineStatusResponse(location string, vmName string) *driver.GetMachineStatusResponse {
	instanceID := DeriveInstanceID(location, vmName)
	return &driver.GetMachineStatusResponse{
		ProviderID: instanceID,
		NodeName:   vmName,
	}
}

func ConstructCreateMachineResponse(location string, vmName string) *driver.CreateMachineResponse {
	instanceID := DeriveInstanceID(location, vmName)
	return &driver.CreateMachineResponse{
		ProviderID: instanceID,
		NodeName:   vmName,
	}
}

func DeriveInstanceID(location, vmName string) string {
	return fmt.Sprintf("azure:///%s/%s", location, vmName)
}

// Helper functions used for driver.DeleteMachine
// ---------------------------------------------------------------------------------------------------------------------

// SkipDeleteMachine checks if ResourceGroup exists. If it does not exist then there is no need to delete any resource as it is assumed that none would exist.
func SkipDeleteMachine(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup string) (bool, error) {
	resGroupAccess, err := factory.GetResourceGroupsAccess(connectConfig)
	if err != nil {
		return false, status.Error(codes.Internal, fmt.Sprintf("failed to create resource group access to process request: [resourceGroup: %s]", resourceGroup))
	}
	resGroupExists, err := accesshelpers.ResourceGroupExists(ctx, resGroupAccess, resourceGroup)
	if err != nil {
		return false, status.Error(codes.Internal, fmt.Sprintf("failed to check if resource group %s exists, Err: %v", resourceGroup, err))
	}
	return !resGroupExists, nil
}

func GetDiskNames(providerSpec api.AzureProviderSpec, vmName string) []string {
	dataDisks := providerSpec.Properties.StorageProfile.DataDisks
	diskNames := make([]string, 0, len(dataDisks)+1)
	diskNames = append(diskNames, utils.CreateOSDiskName(vmName))
	if !utils.IsSliceNilOrEmpty(dataDisks) {
		for _, disk := range dataDisks {
			diskName := utils.CreateDataDiskName(vmName, disk)
			diskNames = append(diskNames, diskName)
		}
	}
	return diskNames
}

func CheckAndDeleteLeftoverNICsAndDisks(ctx context.Context, factory access.Factory, vmName string, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec) error {
	// Gather the names for NIC, OSDisk and Data Disks that needs to be checked for existence and then deleted if they exist.
	resourceGroup := providerSpec.ResourceGroup
	nicName := utils.CreateNICName(vmName)
	diskNames := GetDiskNames(providerSpec, vmName)

	// create NIC and Disks clients
	nicAccess, err := factory.GetNetworkInterfacesAccess(connectConfig)
	if err != nil {
		return err
	}
	disksAccess, err := factory.GetDisksAccess(connectConfig)
	if err != nil {
		return err
	}

	// Create NIC and Disk deletion tasks and run them concurrently.
	tasks := make([]utils.Task, 0, len(diskNames)+1)
	tasks = append(tasks, createNICDeleteTask(resourceGroup, nicName, nicAccess))
	//tasks = append(tasks, d.createDiskDeletionTasks(resourceGroup, diskNames, disksAccess)...)
	tasks = append(tasks, createDisksDeletionTask(resourceGroup, diskNames, disksAccess))
	return errors.Join(utils.RunConcurrently(ctx, tasks, len(tasks))...)
}

func createNICDeleteTask(resourceGroup, nicName string, nicAccess *armnetwork.InterfacesClient) utils.Task {
	return utils.Task{
		Name: fmt.Sprintf("delete-nic-[resourceGroup: %s name: %s]", resourceGroup, nicName),
		Fn: func(ctx context.Context) error {
			return accesshelpers.DeleteNIC(ctx, nicAccess, resourceGroup, nicName)
		},
	}
}

func createDisksDeletionTask(resourceGroup string, diskNames []string, diskAccess *armcompute.DisksClient) utils.Task {
	taskFn := func(ctx context.Context) error {
		var errs []error
		for _, diskName := range diskNames {
			klog.Infof("Deleting disk: [ResourceGroup: %s, DiskName: %s]", resourceGroup, diskName)
			if err := accesshelpers.DeleteDisk(ctx, diskAccess, resourceGroup, diskName); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	return utils.Task{
		Name: fmt.Sprintf("delete-disks-[resourceGroup: %s]", resourceGroup),
		Fn:   taskFn,
	}
}

// Helper functions for driver.CreateMachine
// ---------------------------------------------------------------------------------------------------------------------

func GetSubnet(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec) (*armnetwork.Subnet, error) {
	vnetResourceGroup := providerSpec.ResourceGroup
	if !utils.IsNilOrEmptyStringPtr(providerSpec.SubnetInfo.VnetResourceGroup) {
		vnetResourceGroup = *providerSpec.SubnetInfo.VnetResourceGroup
	}
	subnetAccess, err := factory.GetSubnetAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create subnet access, Err: %v", err))
	}
	subnet, err := accesshelpers.GetSubnet(ctx, subnetAccess, vnetResourceGroup, providerSpec.SubnetInfo.VnetName, providerSpec.SubnetInfo.SubnetName)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get subnet: [ResourceGroup: %s, Name: %s, VNetName: %s], Err: %v", vnetResourceGroup, providerSpec.SubnetInfo.SubnetName, providerSpec.SubnetInfo.VnetName, err))
	}
	return subnet, nil
}

func CreateNICIfNotExists(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec, subnet *armnetwork.Subnet, nicName string) (string, error) {
	nicAccess, err := factory.GetNetworkInterfacesAccess(connectConfig)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("failed to create nic access, Err: %v", err))
	}
	existingNIC, err := accesshelpers.GetNIC(ctx, nicAccess, providerSpec.ResourceGroup, nicName)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("failed to get NIC: [ResourceGroup: %s, Name: %s], Err: %v", providerSpec.ResourceGroup, nicName, err))
	}
	if existingNIC != nil {
		return *existingNIC.ID, nil
	}
	// NIC is not found, create NIC
	nicCreationParams := createNICParams(providerSpec, subnet, nicName)
	nic, err := accesshelpers.CreateNIC(ctx, nicAccess, providerSpec.ResourceGroup, nicCreationParams, nicName)
	if err != nil {
		return "", status.Error(codes.Internal, fmt.Sprintf("failed to create NIC: [ResourceGroup: %s, Name: %s], Err: %v", providerSpec.ResourceGroup, nicName, err))
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

func ProcessVMImageConfiguration(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec, vmName string) (imgRef armcompute.ImageReference, plan *armcompute.Plan, err error) {
	imgRef = getImageReference(providerSpec)
	if vMImageIsMarketPlaceImage(providerSpec) {
		var vmImage *armcompute.VirtualMachineImage
		vmImage, err = getVirtualMachineImage(ctx, factory, connectConfig, providerSpec.Location, imgRef)
		if err != nil {
			return
		}
		if vmImage.Properties != nil && vmImage.Properties.Plan != nil {
			err = checkAndAcceptAgreementIfNotAccepted(ctx, factory, connectConfig, vmName, *vmImage)
			if err != nil {
				return
			}
		}
		plan = &armcompute.Plan{
			Name:      vmImage.Properties.Plan.Name,
			Product:   vmImage.Properties.Plan.Product,
			Publisher: vmImage.Properties.Plan.Publisher,
		}
	}
	return imgRef, plan, nil
}

func getImageReference(providerSpec api.AzureProviderSpec) armcompute.ImageReference {
	imgRefInfo := providerSpec.Properties.StorageProfile.ImageReference

	if !utils.IsEmptyString(imgRefInfo.ID) {
		return armcompute.ImageReference{
			ID: &imgRefInfo.ID,
		}
	}

	if !utils.IsNilOrEmptyStringPtr(imgRefInfo.CommunityGalleryImageID) {
		return armcompute.ImageReference{
			CommunityGalleryImageID: imgRefInfo.CommunityGalleryImageID,
		}
	}

	if !utils.IsNilOrEmptyStringPtr(imgRefInfo.SharedGalleryImageID) {
		return armcompute.ImageReference{
			SharedGalleryImageID: imgRefInfo.SharedGalleryImageID,
		}
	}

	// If we have reached here then, none of ID, CommunityGalleryImageID, SharedGalleryImageID is set.
	// Since the AzureProviderSpec has passed validation its safe to assume that URN is set.
	urnParts := strings.Split(*imgRefInfo.URN, ":")
	return armcompute.ImageReference{
		Publisher: to.Ptr(urnParts[0]),
		Offer:     to.Ptr(urnParts[1]),
		SKU:       to.Ptr(urnParts[2]),
		Version:   to.Ptr(urnParts[3]),
	}
}

func vMImageIsMarketPlaceImage(providerSpec api.AzureProviderSpec) bool {
	imgRef := providerSpec.Properties.StorageProfile.ImageReference
	return imgRef.URN != nil
}

func getVirtualMachineImage(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, location string, imageReference armcompute.ImageReference) (*armcompute.VirtualMachineImage, error) {
	vmImagesAccess, err := factory.GetVirtualMachineImagesAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create image access, Err: %v", err))
	}
	vmImage, err := accesshelpers.GetVMImage(ctx, vmImagesAccess, location, imageReference)
	if err != nil {
		return nil, err
	}
	return vmImage, nil
}

func checkAndAcceptAgreementIfNotAccepted(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, vmName string, vmImage armcompute.VirtualMachineImage) error {
	agreementsAccess, err := factory.GetMarketPlaceAgreementsAccess(connectConfig)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create marketplace agreement access to process request for vm-image: %s, Err: %v", *vmImage.Name, err))
	}
	agreementTerms, err := accesshelpers.GetAgreementTerms(ctx, agreementsAccess, *vmImage.Properties.Plan)
	if err != nil {
		return err
	}
	if agreementTerms.Properties.Accepted == nil || !*agreementTerms.Properties.Accepted {
		err = accesshelpers.AcceptAgreement(ctx, agreementsAccess, *vmImage.Properties.Plan, *agreementTerms)
		if err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("failed to accept agreement for [vmName: %s, vmImage: %s] Err: %v", vmName, *vmImage.Name, err))
		}
	}
	return nil
}

func CreateOrUpdateVM(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec, imageRef armcompute.ImageReference, plan *armcompute.Plan, secret *corev1.Secret, nicID string, vmName string) (*armcompute.VirtualMachine, error) {
	vmAccess, err := factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v", providerSpec.ResourceGroup, vmName, err))
	}
	vmCreationParams, err := createVMCreationParams(providerSpec, imageRef, plan, secret, nicID, vmName)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create virtual machine parameters to create VM: [ResourceGroup: %s, Name: %s], Err: %v", providerSpec.ResourceGroup, vmName, err))
	}
	vm, err := accesshelpers.CreateVirtualMachine(ctx, vmAccess, providerSpec.ResourceGroup, vmCreationParams)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create VirtualMachine: [ResourceGroup: %s, Name: %s], Err: %v", providerSpec.ResourceGroup, vmName, err))
	}
	return vm, nil
}

func createVMCreationParams(providerSpec api.AzureProviderSpec, imageRef armcompute.ImageReference, plan *armcompute.Plan, secret *corev1.Secret, nicID, vmName string) (armcompute.VirtualMachine, error) {
	vmTags := utils.CreateResourceTags(providerSpec.Tags)
	sshConfiguration, err := getSSHConfiguration(providerSpec.Properties.OsProfile.LinuxConfiguration.SSH)
	if err != nil {
		return armcompute.VirtualMachine{}, err
	}

	return armcompute.VirtualMachine{
		Location: to.Ptr(providerSpec.Location),
		Plan:     plan,
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypes(providerSpec.Properties.HardwareProfile.VMSize)),
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: &nicID,
						Properties: &armcompute.NetworkInterfaceReferenceProperties{
							DeleteOption: to.Ptr(armcompute.DeleteOptionsDelete),
							Primary:      to.Ptr(true),
						},
					},
				},
			},
			OSProfile: &armcompute.OSProfile{
				AdminUsername: to.Ptr(providerSpec.Properties.OsProfile.AdminUsername),
				ComputerName:  &vmName,
				CustomData:    to.Ptr(base64.StdEncoding.EncodeToString(secret.Data["userData"])),
				LinuxConfiguration: &armcompute.LinuxConfiguration{
					DisablePasswordAuthentication: to.Ptr(providerSpec.Properties.OsProfile.LinuxConfiguration.DisablePasswordAuthentication),
					SSH:                           sshConfiguration,
				},
			},
			StorageProfile: &armcompute.StorageProfile{
				DataDisks:      getDataDisks(providerSpec.Properties.StorageProfile.DataDisks, vmName),
				ImageReference: &imageRef,
				OSDisk: &armcompute.OSDisk{
					CreateOption: to.Ptr(armcompute.DiskCreateOptionTypes(providerSpec.Properties.StorageProfile.OsDisk.CreateOption)),
					Caching:      to.Ptr(armcompute.CachingTypes(providerSpec.Properties.StorageProfile.OsDisk.Caching)),
					DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
					DiskSizeGB:   pointer.Int32(providerSpec.Properties.StorageProfile.OsDisk.DiskSizeGB),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						StorageAccountType: to.Ptr(armcompute.StorageAccountTypes(providerSpec.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType)),
					},
					Name: to.Ptr(utils.CreateOSDiskName(vmName)),
				},
			},
			AvailabilitySet:        getAvailabilitySet(providerSpec.Properties.AvailabilitySet),
			VirtualMachineScaleSet: getVirtualMachineScaleSet(providerSpec.Properties.VirtualMachineScaleSet),
		},
		Tags:     vmTags,
		Zones:    getZonesFromProviderSpec(providerSpec),
		Name:     &vmName,
		Identity: getVMIdentity(providerSpec.Properties.IdentityID),
	}, nil
}

func getDataDisks(specDataDisks []api.AzureDataDisk, vmName string) []*armcompute.DataDisk {
	var dataDisks []*armcompute.DataDisk
	if utils.IsSliceNilOrEmpty(specDataDisks) {
		return dataDisks
	}
	for _, specDataDisk := range specDataDisks {
		dataDiskName := utils.CreateDataDiskName(vmName, specDataDisk)
		caching := armcompute.CachingTypesNone
		if utils.IsEmptyString(specDataDisk.Caching) {
			caching = armcompute.CachingTypes(specDataDisk.Caching)
		}
		dataDisk := &armcompute.DataDisk{
			CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesEmpty),
			Lun:          specDataDisk.Lun,
			Caching:      to.Ptr(caching),
			DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
			DiskSizeGB:   pointer.Int32(specDataDisk.DiskSizeGB),
			ManagedDisk: &armcompute.ManagedDiskParameters{
				StorageAccountType: to.Ptr(armcompute.StorageAccountTypes(specDataDisk.StorageAccountType)),
			},
			Name: to.Ptr(dataDiskName),
		}
		dataDisks = append(dataDisks, dataDisk)
	}
	return dataDisks
}

func getVMIdentity(specVMIdentityID *string) *armcompute.VirtualMachineIdentity {
	if specVMIdentityID == nil {
		return nil
	}
	return &armcompute.VirtualMachineIdentity{
		Type: to.Ptr(armcompute.ResourceIdentityTypeUserAssigned),
		UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
			*specVMIdentityID: {},
		},
	}
}

func getAvailabilitySet(specAvailabilitySet *api.AzureSubResource) *armcompute.SubResource {
	if specAvailabilitySet == nil {
		return nil
	}
	return &armcompute.SubResource{
		ID: to.Ptr(specAvailabilitySet.ID),
	}
}

func getVirtualMachineScaleSet(specVMSS *api.AzureSubResource) *armcompute.SubResource {
	if specVMSS == nil {
		return nil
	}
	return &armcompute.SubResource{
		ID: to.Ptr(specVMSS.ID),
	}
}

func getSSHConfiguration(sshSpecConfig api.AzureSSHConfiguration) (*armcompute.SSHConfiguration, error) {
	var (
		publicKey string
		err       error
	)
	publicKey = sshSpecConfig.PublicKeys.KeyData
	if utils.IsEmptyString(publicKey) {
		publicKey, err = generateDummyPublicKey()
		if err != nil {
			return nil, err
		}
	}
	return &armcompute.SSHConfiguration{
		PublicKeys: []*armcompute.SSHPublicKey{
			{
				KeyData: to.Ptr(publicKey),
				Path:    to.Ptr(sshSpecConfig.PublicKeys.Path),
			},
		},
	}, nil
}

func generateDummyPublicKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", err
	}
	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(pubKey)
	return string(bytes.Trim(pubKeyBytes, "\x0a")), nil
}

func getZonesFromProviderSpec(spec api.AzureProviderSpec) []*string {
	var zones []*string
	if spec.Properties.Zone != nil {
		zones = append(zones, to.Ptr(strconv.Itoa(*spec.Properties.Zone)))
	}
	return zones
}
