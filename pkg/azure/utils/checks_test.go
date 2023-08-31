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
