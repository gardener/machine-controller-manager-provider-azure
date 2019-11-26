/*
Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

// SampleProviderProviderSpec contains the fields of
// provider spec that the plugin expects
type SampleProviderProviderSpec struct {
	// APIVersion mentions the APIVersion of the object being passed
	APIVersion string

	// TODO: Add the raw extension struct expected while recieving machine operating requests
	// Some dummy examples are mentioned below

	// MachineImageName contains the image name from which machine is to be spawned
	MachineImageName string
	// MachineType constains the type of machine to be spawned
	MachineType string
	// Tags to be placed on the VM
	Tags map[string]string `json:"tags,omitempty"`
}

// Secrets stores the cloud-provider specific sensitive-information.
// +Optional secrets to be passed while performing machine operations on the cloud provider
type Secrets struct {
	// cloud config file (base64 encoded)
	UserData string `json:"userData,omitempty"`
	// CloudCredentials (base64 encoded)
	CloudCredentials string `json:"cloudCredentials,omitempty"`
}
