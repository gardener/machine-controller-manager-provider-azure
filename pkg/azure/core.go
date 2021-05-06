/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package azure contains the cloud provider specific implementations to manage machines
package azure

import (
	"context"
	"fmt"
	"strings"

	api "github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/apis"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/spi"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const (
	// AzureMachineClassKind is the constant representing the AzureMachineClass
	AzureMachineClassKind = "AzureMachineClass"

	// ProviderAzure is the constant representing the Cloud Provider Azure
	ProviderAzure = "Azure"

	// AzureDiskDriverName is the name of the CSI driver for Azure Disk
	AzureDiskDriverName = "disk.csi.azure.com"
)

// VMs maintains a list of VM returned by the provider
// Key refers to the machine-id on the cloud provider
// value refers to the machine-name of the machine object
type VMs map[string]string

// NOTE
//
// The basic working of the controller will work with just implementing the CreateMachine() & DeleteMachine() methods.
// You can first implement these two methods and check the working of the controller.
// Leaving the other methods to NOT_IMPLEMENTED error status.
// Once this works you can implement the rest of the methods.
//
// Also make sure each method return appropriate errors mentioned in `https://github.com/gardener/machine-controller-manager/blob/master/docs/development/machine_error_codes.md`

// CreateMachine handles a machine creation request
// REQUIRED METHOD
//
// REQUEST PARAMETERS (driver.CreateMachineRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM is to be created
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.CreateMachineResponse)
// ProviderID            string                   Unique identification of the VM at the cloud provider. This could be the same/different from req.MachineName.
//                                                ProviderID typically matches with the node.Spec.ProviderID on the node object.
//                                                Eg: gce://project-name/region/vm-ProviderID
// NodeName              string                   Returns the name of the node-object that the VM register's with Kubernetes.
//                                                This could be different from req.MachineName as well
// LastKnownState        string                   (Optional) Last known state of VM during the current operation.
//                                                Could be helpful to continue operations in future requests.
//

// MachinePlugin is the driver struct for holding Azure machine information
type MachinePlugin struct {
	SPI               spi.SessionProviderInterface
	AzureProviderSpec *api.AzureProviderSpec
	Secret            *corev1.Secret
}

// NewAzureDriver returns an empty AzureDriver object
func NewAzureDriver(spi spi.SessionProviderInterface) *MachinePlugin {
	return &MachinePlugin{
		SPI: spi,
	}
}

// CreateMachine . . .
// It is optionally expected by the safety controller to use an identification mechanisms to map the VM Created by a providerSpec.
// These could be done using tag(s)/resource-groups etc.
// This logic is used by safety controller to delete orphan VMs which are not backed by any machine CRD
//
func (d *MachinePlugin) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	// Log messages to track request
	klog.V(2).Infof("Machine creation request has been recieved for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine creation request has been processed for %q", req.Machine.Name)

	// Check if provider in the MachineClass is the provider we support
	if req.MachineClass.Provider != ProviderAzure {
		err := fmt.Errorf("requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, ProviderAzure)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	d.Secret = req.Secret
	virtualMachine, err := d.createVMNicDisk(req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	providerID := encodeMachineID(*virtualMachine.Location, *virtualMachine.Name)
	klog.Infof("Provider ID: %s\nNodeName: %s\n", providerID, *virtualMachine.Name)

	return &driver.CreateMachineResponse{ProviderID: providerID, NodeName: *virtualMachine.Name}, nil
}

// DeleteMachine handles a machine deletion request
//
// REQUEST PARAMETERS (driver.DeleteMachineRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM is to be deleted
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.DeleteMachineResponse)
// LastKnownState        bytes(blob)              (Optional) Last known state of VM during the current operation.
//                                                Could be helpful to continue operations in future requests.
//
func (d *MachinePlugin) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	// Log messages to track delete request
	klog.V(2).Infof("Machine deletion request has been recieved for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine deletion request has been processed for %q", req.Machine.Name)

	// Check if provider in the MachineClass is the provider we support
	if req.MachineClass.Provider != ProviderAzure {
		err := fmt.Errorf("requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, ProviderAzure)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	providerSpec, err := decodeProviderSpecAndSecret(req.MachineClass, req.Secret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	d.AzureProviderSpec = providerSpec
	d.Secret = req.Secret

	var (
		vmName            = strings.ToLower(req.Machine.Name)
		resourceGroupName = providerSpec.ResourceGroup
		nicName           = dependencyNameFromVMName(vmName, nicSuffix)
		diskName          = dependencyNameFromVMName(vmName, diskSuffix)
		dataDiskNames     []string
	)

	clients, err := d.SPI.Setup(d.Secret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Check if the underlying resource group still exists. If not, skip the deletion, as all resources are gone.
	if _, err := clients.GetGroup().Get(ctx, resourceGroupName); err != nil {
		if NotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if providerSpec.Properties.StorageProfile.DataDisks != nil && len(providerSpec.Properties.StorageProfile.DataDisks) > 0 {
		dataDiskNames = getAzureDataDiskNames(providerSpec.Properties.StorageProfile.DataDisks, vmName, dataDiskSuffix)
	}

	err = d.deleteVMNicDisks(ctx, clients, resourceGroupName, vmName, nicName, diskName, dataDiskNames)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &driver.DeleteMachineResponse{}, nil
}

// GetMachineStatus handles a machine get status request
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (driver.GetMachineStatusRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM status needs to be returned
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.GetMachineStatueResponse)
// ProviderID            string                   Unique identification of the VM at the cloud provider. This could be the same/different from req.MachineName.
//                                                ProviderID typically matches with the node.Spec.ProviderID on the node object.
//                                                Eg: gce://project-name/region/vm-ProviderID
// NodeName             string                    Returns the name of the node-object that the VM register's with Kubernetes.
//                                                This could be different from req.MachineName as well
//
// The request should return a NOT_FOUND (5) status error code if the machine is not existing
func (d *MachinePlugin) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("Get request has been recieved for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine get request has been processed successfully for %q", req.Machine.Name)

	// Check if provider in the MachineClass is the provider we support
	if req.MachineClass.Provider != ProviderAzure {
		err := fmt.Errorf("requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, ProviderAzure)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var machineStatusResponse = &driver.GetMachineStatusResponse{}

	listMachineRequest := &driver.ListMachinesRequest{MachineClass: req.MachineClass, Secret: req.Secret}

	machines, err := d.ListMachines(ctx, listMachineRequest)
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
	err = fmt.Errorf("machine '%s' not found", req.Machine.Name)
	return nil, status.Error(codes.NotFound, err.Error())
}

// ListMachines lists all the machines possibilly created by a providerSpec
// Identifying machines created by a given providerSpec depends on the OPTIONAL IMPLEMENTATION LOGIC
// you have used to identify machines created by a providerSpec. It could be tags/resource-groups etc
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (driver.ListMachinesRequest)
// MachineClass          *v1alpha1.MachineClass   MachineClass based on which VMs created have to be listed
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.ListMachinesResponse)
// MachineList           map<string,string>  A map containing the keys as the MachineID and value as the MachineName
//                                           for all machine's who where possibilly created by this ProviderSpec
//
func (d *MachinePlugin) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("List machines request has been recieved for %q", req.MachineClass.Name)
	defer klog.V(2).Infof("List machines request has been recieved for %q", req.MachineClass.Name)

	// Check if provider in the MachineClass is the provider we support
	if req.MachineClass.Provider != ProviderAzure {
		err := fmt.Errorf("requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, ProviderAzure)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	providerSpec, err := decodeProviderSpecAndSecret(req.MachineClass, req.Secret)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	d.AzureProviderSpec = providerSpec
	d.Secret = req.Secret

	var (
		location          = providerSpec.Location
		tags              = providerSpec.Tags
		resourceGroupName = providerSpec.ResourceGroup
		listOfVMs         = make(map[string]string)
	)

	clients, err := d.SPI.Setup(d.Secret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	mergeIntoResult := func(source VMs) {
		for k, v := range source {
			listOfVMs[k] = v
		}
	}

	listOfVMs, err = getRelevantVMs(ctx, clients, resourceGroupName, location, tags)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	mergeIntoResult(listOfVMs)
	klog.Errorf("List of VMs : %s", listOfVMs)

	listOfVMsByNIC, err := getRelevantNICs(ctx, clients, resourceGroupName, location, tags)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	mergeIntoResult(listOfVMsByNIC)

	listOfVMsByDisk, err := getRelevantDisks(ctx, clients, resourceGroupName, location, tags)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	mergeIntoResult(listOfVMsByDisk)

	OnARMAPISuccess(prometheusServiceVM, "VM.List")
	return &driver.ListMachinesResponse{MachineList: listOfVMs}, nil
}

// GetVolumeIDs returns a list of Volume IDs for all PV Specs for whom a provider volume was found
//
// REQUEST PARAMETERS (driver.GetVolumeIDsRequest)
// PVSpecList            []*corev1.PersistentVolumeSpec       PVSpecsList is a list PV specs for whom volume-IDs are required.
//
// RESPONSE PARAMETERS (driver.GetVolumeIDsResponse)
// VolumeIDs             []string                             VolumeIDs is a repeated list of VolumeIDs.
//
func (d *MachinePlugin) GetVolumeIDs(ctx context.Context, req *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("GetVolumeIDs request recieved for %q", req.PVSpecs)
	defer klog.V(2).Infof("GetVolumeIDs request processed successfully for %q", req.PVSpecs)

	names := []string{}
	specs := req.PVSpecs

	for i := range specs {
		spec := specs[i]
		if spec.AzureDisk != nil {
			name := spec.AzureDisk.DiskName
			names = append(names, name)
		} else if spec.CSI != nil && spec.CSI.Driver == AzureDiskDriverName && spec.CSI.VolumeHandle != "" {
			name := spec.CSI.VolumeHandle
			names = append(names, name)
		}
	}

	return &driver.GetVolumeIDsResponse{VolumeIDs: names}, nil
}

// GenerateMachineClassForMigration helps in migration of one kind of machineClass CR to another kind.
// For instance an machineClass custom resource of `AzureMachineClass` to `MachineClass`.
// Implement this functionality only if something like this is desired in your setup.
// If you don't require this functionality leave is as is. (return Unimplemented)
//
// The following are the tasks typically expected out of this method
// 1. Validate if the incoming classSpec is valid one for migration (e.g. has the right kind).
// 2. Migrate/Copy over all the fields/spec from req.ProviderSpecificMachineClass to req.MachineClass
// For an example refer
//		https://github.com/prashanth26/machine-controller-manager-provider-gcp/blob/migration/pkg/gcp/machine_controller.go#L222-L233
//
// REQUEST PARAMETERS (driver.GenerateMachineClassForMigration)
// ProviderSpecificMachineClass    interface{}                             ProviderSpecificMachineClass is provider specfic machine class object (E.g. AzureMachineClass). Typecasting is required here.
// MachineClass 				   *v1alpha1.MachineClass                  MachineClass is the machine class object that is to be filled up by this method.
// ClassSpec                       *v1alpha1.ClassSpec                     Somemore classSpec details useful while migration.
//
// RESPONSE PARAMETERS (driver.GenerateMachineClassForMigration)
// NONE
//
func (d *MachinePlugin) GenerateMachineClassForMigration(ctx context.Context, req *driver.GenerateMachineClassForMigrationRequest) (*driver.GenerateMachineClassForMigrationResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("MigrateMachineClass request has been recieved for %q", req.ClassSpec)
	defer klog.V(2).Infof("MigrateMachineClass request has been processed successfully for %q", req.ClassSpec)

	azureMachineClass := req.ProviderSpecificMachineClass.(*v1alpha1.AzureMachineClass)

	// Check if incoming CR is valid CR for migration
	// In this case, the MachineClassKind to be matching
	if req.ClassSpec.Kind != AzureMachineClassKind {
		return nil, status.Error(codes.Internal, "Migration cannot be done for this machineClass kind")
	}

	return &driver.GenerateMachineClassForMigrationResponse{}, fillUpMachineClass(azureMachineClass, req.MachineClass)
}
