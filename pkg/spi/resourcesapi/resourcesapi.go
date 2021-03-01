/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
SPDX-License-Identifier: Apache-2.0
*/

package resourcesapi

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
)

// GroupsClientAPI TODO
type GroupsClientAPI interface {
	Get(ctx context.Context, resourceGroupName string) (result resources.Group, err error)
}
