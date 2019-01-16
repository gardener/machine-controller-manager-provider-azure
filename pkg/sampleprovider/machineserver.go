/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This file was copied and modified from the kubernetes-csi/drivers project
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/sampleprovider/machineserver.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package sampleprovider

import (
	"fmt"

	cmicommon "github.com/gardener/machine-controller-manager-provider-sampleprovider/pkg/cmi-common"
	api "github.com/gardener/machine-controller-manager-provider-sampleprovider/pkg/sampleprovider/apis"
	"github.com/gardener/machine-spec/lib/go/cmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MachineServer contains the machine server info
type MachineServer struct {
	*cmicommon.DefaultMachineServer
}

// CreateMachine is used to create a new machine
func (ms *MachineServer) CreateMachine(ctx context.Context, req *cmi.CreateMachineRequest) (*cmi.CreateMachineResponse, error) {
	// Sample code to access provider spec
	// Delete the 4 following line in the controller implementation
	dummyObject := api.SampleProviderProviderSpec{
		APIVersion: "v1alpha1",
	}
	fmt.Println("APIVersion of object ", dummyObject.APIVersion)

	return nil, status.Error(codes.Unimplemented, "")
}

// DeleteMachine is used to delete a machine
func (ms *MachineServer) DeleteMachine(ctx context.Context, req *cmi.DeleteMachineRequest) (*cmi.DeleteMachineResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListMachines is the default method used to list machines
// Returns a VM matching the machineID, but when the machineID is an empty string
// then it returns all matching instances in terms of map[string]string
func (ms *MachineServer) ListMachines(ctx context.Context, req *cmi.ListMachinesRequest) (*cmi.ListMachinesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
