/*
SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package azure contains the cloud provider specific implementations to manage machines
package azure

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider Suite")
}
