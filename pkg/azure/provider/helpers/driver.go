// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	accesserrors "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api/validation"
	"golang.org/x/crypto/ssh"
	"k8s.io/utils/pointer"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	accesshelpers "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/helpers"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// ExtractProviderSpecAndConnectConfig extracts api.AzureProviderSpec from mcc and access.ConnectConfig from secret.
func ExtractProviderSpecAndConnectConfig(mcc *v1alpha1.MachineClass, secret *corev1.Secret) (api.AzureProviderSpec, access.ConnectConfig, error) {
	var (
		err           error
		providerSpec  api.AzureProviderSpec
		connectConfig access.ConnectConfig
	)
	// validate provider Spec provider. Exit early if it is not azure.
	if err = validation.ValidateMachineClassProvider(mcc); err != nil {
		return api.AzureProviderSpec{}, access.ConnectConfig{}, err
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

// ConstructMachineListResponse constructs response for driver.ListMachines method.
func ConstructMachineListResponse(location string, vmNames []string) *driver.ListMachinesResponse {
	listMachineRes := driver.ListMachinesResponse{}
	instanceIDToVMNameMap := make(map[string]string, len(vmNames))
	if len(vmNames) == 0 {
		return &listMachineRes
	}
	for _, vmName := range vmNames {
		instanceIDToVMNameMap[DeriveInstanceID(location, vmName)] = vmName
	}
	listMachineRes.MachineList = instanceIDToVMNameMap
	return &listMachineRes
}

// ConstructGetMachineStatusResponse constructs response for driver.GetMachineStatus method.
func ConstructGetMachineStatusResponse(location string, vmName string) *driver.GetMachineStatusResponse {
	instanceID := DeriveInstanceID(location, vmName)
	return &driver.GetMachineStatusResponse{
		ProviderID: instanceID,
		NodeName:   vmName,
	}
}

// ConstructCreateMachineResponse constructs response for driver.CreateMachine method.
func ConstructCreateMachineResponse(location string, vmName string) *driver.CreateMachineResponse {
	instanceID := DeriveInstanceID(location, vmName)
	return &driver.CreateMachineResponse{
		ProviderID: instanceID,
		NodeName:   vmName,
	}
}

// DeriveInstanceID creates an instance ID from location and VM name.
func DeriveInstanceID(location, vmName string) string {
	return fmt.Sprintf("azure:///%s/%s", location, vmName)
}

// Helper functions used for driver.DeleteMachine
// ---------------------------------------------------------------------------------------------------------------------

// SkipDeleteMachine checks if ResourceGroup exists. If it does not exist then there is no need to delete any resource as it is assumed that none would exist.
func SkipDeleteMachine(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, resourceGroup string) (bool, error) {
	resGroupAccess, err := factory.GetResourceGroupsAccess(connectConfig)
	if err != nil {
		return false, status.WrapError(codes.Internal, fmt.Sprintf("failed to create ResourceGroup access to process request: [ResourceGroup: %s]", resourceGroup), err)
	}
	resGroupExists, err := accesshelpers.ResourceGroupExists(ctx, resGroupAccess, resourceGroup)
	if err != nil {
		return false, status.WrapError(codes.Internal, fmt.Sprintf("failed to check if ResourceGroup %s exists, Err: %v", resourceGroup, err), err)
	}
	return !resGroupExists, nil
}

// GetDiskNames creates disk names for all configured OSDisk and DataDisk in the provider spec.
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

// CheckAndDeleteLeftoverNICsAndDisks creates tasks for NIC and DISK deletion and runs them concurrently. It waits for them to complete and then returns a consolidated error if there is any.
// This method will be called when these resources are left without an associated VM.
func CheckAndDeleteLeftoverNICsAndDisks(ctx context.Context, factory access.Factory, vmName string, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec) error {
	// Gather the names for NIC, OSDisk and Data Disks that needs to be checked for existence and then deleted if they exist.
	resourceGroup := providerSpec.ResourceGroup
	nicName := utils.CreateNICName(vmName)
	diskNames := GetDiskNames(providerSpec, vmName)

	// create NIC and Disks clients
	nicAccess, err := factory.GetNetworkInterfacesAccess(connectConfig)
	if err != nil {
		return status.WrapError(codes.Internal, fmt.Sprintf("Failed to create nic access for VM: [ResourceGroup: %s, Name: %s], Err: %v", resourceGroup, vmName, err), err)
	}
	disksAccess, err := factory.GetDisksAccess(connectConfig)
	if err != nil {
		return status.WrapError(codes.Internal, fmt.Sprintf("Failed to create disk access for VM: [ResourceGroup: %s, Name: %s], Err: %v", resourceGroup, vmName, err), err)
	}

	// Create NIC and Disk deletion tasks and run them concurrently.
	tasks := make([]utils.Task, 0, len(diskNames)+1)
	tasks = append(tasks, createNICDeleteTask(resourceGroup, nicName, nicAccess))
	tasks = append(tasks, createDisksDeletionTasks(resourceGroup, diskNames, disksAccess)...)
	combinedErr := errors.Join(utils.RunConcurrently(ctx, tasks, 2)...)
	if combinedErr != nil {
		return status.WrapError(codes.Internal, fmt.Sprintf("Errors during deletion of NIC/Disks associated to VM: [ResourceGroup: %s, Name: %s], Err: %v", resourceGroup, vmName, err), combinedErr)
	}
	return nil
}

// UpdateCascadeDeleteOptions updates the VirtualMachine properties and sets cascade delete options for NIC's and DISK's if it is not already set.
// Once that is set then it deletes the VM. This will ensure that no separate calls to delete each NIC and DISK are made as they will get deleted along with the VM in one single atomic call.
func UpdateCascadeDeleteOptions(ctx context.Context, vmAccess *armcompute.VirtualMachinesClient, resourceGroup string, vm *armcompute.VirtualMachine) error {
	vmName := *vm.Name
	if canUpdateVirtualMachine(vm) {
		vmUpdateParams := computeDeleteOptionUpdatesForNICsAndDisksIfRequired(resourceGroup, vm)
		if vmUpdateParams != nil {
			// update the VM and set cascade delete on NIC and Disks (OSDisk and DataDisks) if not already set and then trigger VM deletion.
			klog.V(4).Infof("Updating cascade deletion options for VM: [ResourceGroup: %s, Name: %s] resources", resourceGroup, vmName)
			err := accesshelpers.SetCascadeDeleteForNICsAndDisks(ctx, vmAccess, resourceGroup, vmName, vmUpdateParams)
			if err != nil {
				return status.WrapError(codes.Internal, fmt.Sprintf("Failed to update cascade delete of associated resources for VM: [ResourceGroup: %s, Name: %s], Err: %v", resourceGroup, vmName, err), err)
			}
		}
	} else {
		return status.New(codes.Internal, fmt.Sprintf("Cannot update VM: [ResourceGroup: %s, Name: %s]. Either the VM has provisionState set to Failed or there are one or more data disks that are marked for detachment, update call to this VM will fail. Skipping the update call.", resourceGroup, vmName))
	}
	return nil
}

// DeleteVirtualMachine deletes the VirtualMachine, if there is any error it will wrap it into a status.Status error.
func DeleteVirtualMachine(ctx context.Context, vmAccess *armcompute.VirtualMachinesClient, resourceGroup string, vmName string) error {
	klog.Infof("Deleting VM: [ResourceGroup: %s, Name: %s]", resourceGroup, vmName)
	err := accesshelpers.DeleteVirtualMachine(ctx, vmAccess, resourceGroup, vmName)
	if err != nil {
		return status.WrapError(codes.Internal, fmt.Sprintf("Failed to delete VM: [ResourceGroup: %s, Name: %s], Err: %v", resourceGroup, vmName, err), err)
	}
	return nil
}

// IsVirtualMachineInTerminalState checks if the provisioningState of the VM is set to Failed.
func IsVirtualMachineInTerminalState(vm *armcompute.VirtualMachine) bool {
	return vm.Properties != nil && vm.Properties.ProvisioningState != nil && strings.ToLower(*vm.Properties.ProvisioningState) == strings.ToLower(utils.ProvisioningStateFailed)
}

func canUpdateVirtualMachine(vm *armcompute.VirtualMachine) bool {
	return IsVirtualMachineInTerminalState(vm) || utils.DataDisksMarkedForDetachment(vm)
}

// computeDeleteOptionUpdatesForNICsAndDisksIfRequired computes changes required to set cascade delete options for NICs, OSDisk and DataDisks.
// If there are no changes then a nil is returned. If there are changes then delta changes are captured in armcompute.VirtualMachineUpdate
func computeDeleteOptionUpdatesForNICsAndDisksIfRequired(resourceGroup string, vm *armcompute.VirtualMachine) *armcompute.VirtualMachineUpdate {
	var (
		vmUpdateParams       *armcompute.VirtualMachineUpdate
		updatedNicReferences []*armcompute.NetworkInterfaceReference
		updatedDataDisks     []*armcompute.DataDisk
		updatedOSDisk        *armcompute.OSDisk
		vmName               = *vm.Name
	)

	// Return early if VM does not have any properties set. This should ideally never happen.
	if vm.Properties == nil {
		klog.Errorf("Weird, but it seems that the VM: [ResourceGroup: %s, Name: %s] does not have Properties set, skipping computing cascade delete update computation for this VM", resourceGroup, vmName)
		return vmUpdateParams
	}

	updatedNicReferences = getNetworkInterfaceReferencesToUpdate(vm.Properties.NetworkProfile)
	updatedOSDisk = getOSDiskToUpdate(vm.Properties.StorageProfile)
	updatedDataDisks = getDataDisksToUpdate(vm.Properties.StorageProfile)

	// If there are no updates on NIC(s), OSDisk and DataDisk(s) then just return early.
	if utils.IsSliceNilOrEmpty(updatedNicReferences) && updatedOSDisk == nil && utils.IsSliceNilOrEmpty(updatedDataDisks) {
		klog.Infof("All configured NICs, OSDisk and DataDisks have cascade delete already set for VM: [ResourceGroup: %s, Name: %s]", resourceGroup, vmName)
		return vmUpdateParams
	}

	vmUpdateParams = &armcompute.VirtualMachineUpdate{
		Properties: &armcompute.VirtualMachineProperties{
			StorageProfile: &armcompute.StorageProfile{},
		},
	}

	if !utils.IsSliceNilOrEmpty(updatedNicReferences) {
		klog.Infof("Identified #%d NICs requiring DeleteOption updates for VM: [ResourceGroup: %s, Name: %s]", len(updatedNicReferences), resourceGroup, vmName)
		vmUpdateParams.Properties.NetworkProfile = &armcompute.NetworkProfile{
			NetworkInterfaces: updatedNicReferences,
		}
	}
	if updatedOSDisk != nil {
		klog.Infof("Identified OSDisk: %s requiring DeleteOption update for VM: [ResourceGroup: %s, Name: %s]", *updatedOSDisk.Name, resourceGroup, vmName)
		vmUpdateParams.Properties.StorageProfile.OSDisk = updatedOSDisk
	}
	if !utils.IsSliceNilOrEmpty(updatedDataDisks) {
		vmUpdateParams.Properties.StorageProfile.DataDisks = updatedDataDisks
	}

	return vmUpdateParams
}

// getNetworkInterfaceReferencesToUpdate checks if there are still NICs which do not have cascade delete set. These are captured and changed
// NetworkInterfaceReference's are then returned with cascade delete option set.
func getNetworkInterfaceReferencesToUpdate(networkProfile *armcompute.NetworkProfile) []*armcompute.NetworkInterfaceReference {
	if networkProfile == nil || utils.IsSliceNilOrEmpty(networkProfile.NetworkInterfaces) {
		return nil
	}
	updatedNicRefs := make([]*armcompute.NetworkInterfaceReference, 0, len(networkProfile.NetworkInterfaces))
	for _, nicRef := range networkProfile.NetworkInterfaces {
		updatedNicRef := &armcompute.NetworkInterfaceReference{ID: nicRef.ID}
		if !isNicCascadeDeleteSet(nicRef) {
			if updatedNicRef.Properties == nil {
				updatedNicRef.Properties = &armcompute.NetworkInterfaceReferenceProperties{}
			}
			updatedNicRef.Properties.DeleteOption = to.Ptr(armcompute.DeleteOptionsDelete)
			updatedNicRefs = append(updatedNicRefs, updatedNicRef)
		}
	}
	return updatedNicRefs
}

// isNicCascadeDeleteSet checks if for a given NIC, cascade deletion option is set.
func isNicCascadeDeleteSet(nicRef *armcompute.NetworkInterfaceReference) bool {
	if nicRef.Properties == nil {
		return false
	}
	deleteOption := nicRef.Properties.DeleteOption
	return deleteOption != nil && *deleteOption == armcompute.DeleteOptionsDelete
}

// getOSDiskToUpdate checks if cascade delete option is set on OSDisk, if it is not then it will set it and return the
// updated OSDisk else it will return nil.
func getOSDiskToUpdate(storageProfile *armcompute.StorageProfile) *armcompute.OSDisk {
	var updatedOSDisk *armcompute.OSDisk
	if storageProfile != nil && storageProfile.OSDisk != nil {
		existingOSDisk := storageProfile.OSDisk
		existingDeleteOption := existingOSDisk.DeleteOption
		if existingDeleteOption == nil || *existingDeleteOption != armcompute.DiskDeleteOptionTypesDelete {
			updatedOSDisk = &armcompute.OSDisk{
				Name:         existingOSDisk.Name,
				DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
			}
		}
	}
	return updatedOSDisk
}

// getDataDisksToUpdate checks if cascade delete option set for all DataDisks attached to the Virtual machine.
// All data disks that do not have that set, it will set the appropriate DeleteOption and return the updated
// DataDisks else it will return nil
func getDataDisksToUpdate(storageProfile *armcompute.StorageProfile) []*armcompute.DataDisk {
	var updatedDataDisks []*armcompute.DataDisk
	if storageProfile != nil && !utils.IsSliceNilOrEmpty(storageProfile.DataDisks) {
		updatedDataDisks = make([]*armcompute.DataDisk, 0, len(storageProfile.DataDisks))
		for _, dataDisk := range storageProfile.DataDisks {
			if dataDisk.DeleteOption == nil || *dataDisk.DeleteOption != armcompute.DiskDeleteOptionTypesDelete {
				updatedDataDisk := &armcompute.DataDisk{
					Lun:          dataDisk.Lun,
					DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
					Name:         dataDisk.Name,
				}
				updatedDataDisks = append(updatedDataDisks, updatedDataDisk)
			}
		}
	}
	return updatedDataDisks
}

func createNICDeleteTask(resourceGroup, nicName string, nicAccess *armnetwork.InterfacesClient) utils.Task {
	return utils.Task{
		Name: fmt.Sprintf("delete-nic-[resourceGroup: %s name: %s]", resourceGroup, nicName),
		Fn: func(ctx context.Context) error {
			klog.Infof("Attempting to delete nic: [ResourceGroup: %s, NicName: %s] if it exists", resourceGroup, nicName)
			return accesshelpers.DeleteNIC(ctx, nicAccess, resourceGroup, nicName)
		},
	}
}

func createDisksDeletionTasks(resourceGroup string, diskNames []string, diskAccess *armcompute.DisksClient) []utils.Task {
	tasks := make([]utils.Task, 0, len(diskNames))
	for _, diskName := range diskNames {
		diskName := diskName // TODO: remove this once https://github.com/golang/go/wiki/LoopvarExperiment becomes part of 1.21.x
		taskFn := func(ctx context.Context) error {
			klog.Infof("Attempting to delete disk: [ResourceGroup: %s, DiskName: %s] if it exists", resourceGroup, diskName)
			return accesshelpers.DeleteDisk(ctx, diskAccess, resourceGroup, diskName)
		}
		tasks = append(tasks, utils.Task{
			Name: fmt.Sprintf("delete-disk-[resourceGroup: %s name: %s]", resourceGroup, diskName),
			Fn:   taskFn,
		})
	}
	return tasks
}

// Helper functions for driver.CreateMachine
// ---------------------------------------------------------------------------------------------------------------------

// GetSubnet gets the subnet for the subnet configuration in the provider config.
func GetSubnet(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec) (*armnetwork.Subnet, error) {
	vnetResourceGroup := providerSpec.ResourceGroup
	if !utils.IsNilOrEmptyStringPtr(providerSpec.SubnetInfo.VnetResourceGroup) {
		vnetResourceGroup = *providerSpec.SubnetInfo.VnetResourceGroup
	}
	subnetAccess, err := factory.GetSubnetAccess(connectConfig)
	if err != nil {
		return nil, status.WrapError(codes.Internal, fmt.Sprintf("failed to create subnet access, Err: %v", err), err)
	}
	subnet, err := accesshelpers.GetSubnet(ctx, subnetAccess, vnetResourceGroup, providerSpec.SubnetInfo.VnetName, providerSpec.SubnetInfo.SubnetName)
	if err != nil {
		return nil, status.WrapError(codes.Internal, fmt.Sprintf("failed to get subnet: [ResourceGroup: %s, Name: %s, VNetName: %s], Err: %v", vnetResourceGroup, providerSpec.SubnetInfo.SubnetName, providerSpec.SubnetInfo.VnetName, err), err)
	}
	klog.Infof("Retrieved Subnet: [ResourceGroup: %s, Name:%s, VNetName: %s]", vnetResourceGroup, providerSpec.SubnetInfo.SubnetName, providerSpec.SubnetInfo.VnetName)
	return subnet, nil
}

// CreateNICIfNotExists creates a NIC if it does not exist.
func CreateNICIfNotExists(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec, subnet *armnetwork.Subnet, nicName string) (string, error) {
	nicAccess, err := factory.GetNetworkInterfacesAccess(connectConfig)
	if err != nil {
		return "", status.WrapError(codes.Internal, fmt.Sprintf("failed to create nic access, Err: %v", err), err)
	}
	resourceGroup := providerSpec.ResourceGroup
	existingNIC, err := accesshelpers.GetNIC(ctx, nicAccess, resourceGroup, nicName)
	if err != nil {
		return "", status.WrapError(codes.Internal, fmt.Sprintf("Failed to get NIC: [ResourceGroup: %s, Name: %s], Err: %v", resourceGroup, nicName, err), err)
	}
	if existingNIC != nil {
		klog.Infof("[ResourceGroup: %s, NIC: [Name: %s, ID: %s]] exists, will skip creation of the NIC", resourceGroup, nicName, *existingNIC.ID)
		return *existingNIC.ID, nil
	}
	// NIC is not found, create NIC
	nicCreationParams := createNICParams(providerSpec, subnet, nicName)
	nic, err := accesshelpers.CreateNIC(ctx, nicAccess, providerSpec.ResourceGroup, nicCreationParams, nicName)
	if err != nil {
		return "", status.WrapError(codes.Internal, fmt.Sprintf("failed to create NIC: [ResourceGroup: %s, Name: %s], Err: %v", providerSpec.ResourceGroup, nicName, err), err)
	}
	klog.Infof("Successfully created NIC: [ResourceGroup: %s, NIC: [Name: %s, ID: %s]]", resourceGroup, nicName, *nic.ID)
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

// ProcessVMImageConfiguration gets the image configuration from provider spec. If the VM image configured is a marketplace image then it will additionally do the following:
// 1. Gets the VM image. If the image does not exist then it will return an error.
// 2. From the VM Image it checks if there is a plan.
// 3. If there is a plan then it will check if there is an existing agreement for this plan. If an agreement does not exist then it will return an error.
// 4. If the agreement has not been accepted yet then it will accept the agreement and update the agreement. If that fails then it will return an error.
func ProcessVMImageConfiguration(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec, vmName string) (imgRef armcompute.ImageReference, plan *armcompute.Plan, err error) {
	imgRef = getImageReference(providerSpec)
	isMarketPlaceImage := providerSpec.Properties.StorageProfile.ImageReference.URN != nil
	if isMarketPlaceImage {
		var vmImage *armcompute.VirtualMachineImage
		vmImage, err = getVirtualMachineImage(ctx, factory, connectConfig, providerSpec.Location, imgRef)
		if err != nil {
			return
		}
		klog.Infof("Retrieved VM Image: [VMName: %s, ID: %s]", vmName, *vmImage.ID)
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

func getVirtualMachineImage(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, location string, imageReference armcompute.ImageReference) (*armcompute.VirtualMachineImage, error) {
	vmImagesAccess, err := factory.GetVirtualMachineImagesAccess(connectConfig)
	if err != nil {
		return nil, status.WrapError(codes.Internal, fmt.Sprintf("Failed to create image access, Err: %v", err), err)
	}
	vmImage, err := accesshelpers.GetVMImage(ctx, vmImagesAccess, location, imageReference)
	if err != nil {
		if accesserrors.IsNotFoundAzAPIError(err) {
			return nil, status.WrapError(codes.NotFound, fmt.Sprintf("VM Image %v does not exist", imageReference), err)
		}
		return nil, status.WrapError(codes.Internal, fmt.Sprintf("Failed to retrieve VM Image: %v", imageReference), err)
	}
	return vmImage, nil
}

// checkAndAcceptAgreementIfNotAccepted checks if an agreement exists. If it does not exist it returns an error. If it does exist and agreement has not been accepted then it will accept the
// agreement and if that fails then it will return an error.
// NOTE: Today agreement needs to be created by the customer. However, if the agreement has not been accepted then we accept the agreement on behalf of the customer. This is not really ideal and is only done
// for ease of consumption of garden-linux image. This should be done till the point garden-linux VM image is eventually made available as a community image. As of today community gallery is a alpha feature.
// Once it becomes GA then we should shift to using community image for garden-linux. Then we should remove the code which accepts the agreement on behalf of the customer.
func checkAndAcceptAgreementIfNotAccepted(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, vmName string, vmImage armcompute.VirtualMachineImage) error {
	agreementsAccess, err := factory.GetMarketPlaceAgreementsAccess(connectConfig)
	if err != nil {
		return status.WrapError(codes.Internal, fmt.Sprintf("Failed to create marketplace agreement access to process request for vm-image: %s, Err: %v", *vmImage.Name, err), err)
	}
	plan := *vmImage.Properties.Plan
	agreementTerms, err := accesshelpers.GetAgreementTerms(ctx, agreementsAccess, plan)
	if err != nil {
		if accesserrors.IsNotFoundAzAPIError(err) {
			return status.WrapError(codes.NotFound, fmt.Sprintf("Marketplace Image Agreement for Plan [Name: %s, Product: %s, Publisher: %s] does not exist", *plan.Name, *plan.Product, *plan.Publisher), err)
		}
		return status.WrapError(codes.Internal, fmt.Sprintf("Failed to retrieve Marketplace Image Agreement for Plan [Name: %s, Product: %s, Publisher: %s]", *plan.Name, *plan.Product, *plan.Publisher), err)

	}
	klog.Infof("Retrieved Marketplace Image Agreement for Plan [Name: %s, Product: %s, Publisher: %s]", *plan.Name, *plan.Product, *plan.Publisher)
	if agreementTerms.Properties.Accepted == nil || !*agreementTerms.Properties.Accepted {
		err = accesshelpers.AcceptAgreement(ctx, agreementsAccess, *vmImage.Properties.Plan, *agreementTerms)
		if err != nil {
			return status.WrapError(codes.Internal, fmt.Sprintf("Failed to accept agreement for [VMName: %s, VMImageID: %s, Plan: {Name: %s, Product: %s, Publisher: %s}] Err: %v", vmName, *vmImage.ID, *plan.Name, *plan.Product, *plan.Publisher, err), err)
		}
	}
	klog.Infof("Successfully validated/updated agreement terms as accepted for [VMName: %s, VMImage: %s, AgreementID: %s]", vmName, *vmImage.ID, *agreementTerms.ID)
	return nil
}

// CreateVM gathers the VM creation parameters and invokes a call to create or update the VM.
func CreateVM(ctx context.Context, factory access.Factory, connectConfig access.ConnectConfig, providerSpec api.AzureProviderSpec, imageRef armcompute.ImageReference, plan *armcompute.Plan, secret *corev1.Secret, nicID string, vmName string) (*armcompute.VirtualMachine, error) {
	vmAccess, err := factory.GetVirtualMachinesAccess(connectConfig)
	if err != nil {
		return nil, status.WrapError(codes.Internal, fmt.Sprintf("Failed to create virtual machine access to process request: [resourceGroup: %s, vmName: %s], Err: %v", providerSpec.ResourceGroup, vmName, err), err)
	}
	vmCreationParams, err := createVMCreationParams(providerSpec, imageRef, plan, secret, nicID, vmName)
	if err != nil {
		return nil, status.WrapError(codes.Internal, fmt.Sprintf("Failed to create virtual machine parameters to create VM: [ResourceGroup: %s, Name: %s], Err: %v", providerSpec.ResourceGroup, vmName, err), err)
	}
	vm, err := accesshelpers.CreateVirtualMachine(ctx, vmAccess, providerSpec.ResourceGroup, vmCreationParams)
	if err != nil {
		errCode := accesserrors.GetMatchingErrorCode(err)
		return nil, status.WrapError(errCode, fmt.Sprintf("Failed to create VirtualMachine: [ResourceGroup: %s, Name: %s], Err: %v", providerSpec.ResourceGroup, vmName, err), err)
	}
	klog.Infof("Successfully created VM: [ResourceGroup: %s, Name: %s]", providerSpec.ResourceGroup, vmName)
	return vm, nil
}

// LogVMCreation is a convenience method which helps to extract relevant details from the created virtual machine and logs it.
// Today the azure create VM call is atomic only w.r.t creation of VM, OSDisk, DataDisk(s). NIC still has to be created prior to creation of the VM.
// Therefore, this method produces a log which also prints the OSDisk, DataDisks that are created (which helps in traceability). For completeness it
// also prints the NIC that now gets associated to this VM.
func LogVMCreation(location, resourceGroup string, vm *armcompute.VirtualMachine) {
	msgBuilder := strings.Builder{}
	vmName := *vm.Name
	msgBuilder.WriteString(fmt.Sprintf("Successfully create Machine in [Location: %s, ResourceGroup: %s] with the following resources:\n", location, resourceGroup))
	msgBuilder.WriteString(fmt.Sprintf("VirtualMachine: [ID: %s, Name: %s]\n", *vm.ID, vmName))
	if !utils.IsSliceNilOrEmpty(vm.Properties.NetworkProfile.NetworkInterfaces) {
		nic := vm.Properties.NetworkProfile.NetworkInterfaces[0]
		msgBuilder.WriteString(fmt.Sprintf("NIC: [ID: %s, Name: %s]\n", *nic.ID, utils.CreateNICName(vmName)))
	}
	if vm.Properties.StorageProfile.OSDisk != nil {
		msgBuilder.WriteString(fmt.Sprintf("OSDisk: %s\n", *vm.Properties.StorageProfile.OSDisk.Name))
	}
	if !utils.IsSliceNilOrEmpty(vm.Properties.StorageProfile.DataDisks) {
		msgBuilder.WriteString("DataDisks: [ ")
		for _, dataDisk := range vm.Properties.StorageProfile.DataDisks {
			msgBuilder.WriteString(fmt.Sprintf("{Name: %s} ", *dataDisk.Name))
		}
		msgBuilder.WriteString(" ]")
	}
	klog.Infof(msgBuilder.String())
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
