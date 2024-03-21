// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
