// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

func TestIsEmptyString(t *testing.T) {
	table := []struct {
		description    string
		strValue       string
		expectedResult bool
	}{
		{"empty string should result true", "", true},
		{"string with only spaces should result true", "  ", true},
		{"non-empty string should result false", "bingo", false},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry)
		g.Expect(IsEmptyString(entry.strValue)).To(Equal(entry.expectedResult))
	}
}

func TestIsNilOrEmptyStringPtr(t *testing.T) {
	table := []struct {
		description    string
		strValue       *string
		expectedResult bool
	}{
		{"empty string should result true", pointer.String(""), true},
		{"string with only spaces should result true", pointer.String("  "), true},
		{"nil string should result true", nil, true},
		{"non-empty string should result false", pointer.String("bingo"), false},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry)
		g.Expect(IsNilOrEmptyStringPtr(entry.strValue)).To(Equal(entry.expectedResult))
	}
}

func TestIsSliceNilOrEmpty(t *testing.T) {
	table := []struct {
		description    string
		slice          []string
		expectedResult bool
	}{
		{"nil slice should result true", nil, true},
		{"empty slice should result true", []string{}, true},
		{"slice with atleast one entry should result false", []string{"bingo"}, false},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry)
		g.Expect(IsSliceNilOrEmpty(entry.slice)).To(Equal(entry.expectedResult))
	}
}
