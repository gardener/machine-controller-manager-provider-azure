/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

// Package azure contains the cloud provider specific implementations to manage machines
package azure

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {

	Describe("PublicKey", func() {

		Describe("#Generate", func() {
			It("should properly generate PublicKey string", func() {
				publicKey, err := generatePublicKey()
				Expect(err).NotTo(HaveOccurred())

				Expect(publicKey).NotTo(Equal(""))
			})
		})
	})
})
