/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package spi

import (
	corev1 "k8s.io/api/core/v1"
)

// SessionProviderInterface provides an interface to deal with cloud provider session
// Example interfaces are listed below.
type SessionProviderInterface interface {
	Setup(cloudConfig *corev1.Secret) (AzureDriverClientsInterface, error)
}
