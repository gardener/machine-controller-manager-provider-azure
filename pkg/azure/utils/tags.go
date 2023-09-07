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

package utils

import "github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

const (
	// ClusterTagPrefix is a prefix for a mandatory cluster tag on resources
	ClusterTagPrefix = "kubernetes.io-cluster-"
	// RoleTagPrefix is a prefix for a mandatory role tag on resources
	RoleTagPrefix = "kubernetes.io-role-"
)

// CreateResourceTags changes the tag value to be a pointer to string. Azure APIs require tags to be represented as map[string]*string
func CreateResourceTags(tags map[string]string) map[string]*string {
	vmTags := make(map[string]*string, len(tags))
	for k, v := range tags {
		vmTags[k] = to.Ptr(v)
	}
	return vmTags
}
