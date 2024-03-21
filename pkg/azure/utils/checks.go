// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import "strings"

// IsEmptyString trims the spaces around the string and checks its length.
// If it is 0 then it will return true else it will return false
func IsEmptyString(s string) bool {
	if len(strings.TrimSpace(s)) == 0 {
		return true
	}
	return false
}

// IsNilOrEmptyStringPtr returns true if the string pointer is nil or the return value of IsEmptyString(s).
func IsNilOrEmptyStringPtr(s *string) bool {
	if s == nil {
		return true
	}
	return IsEmptyString(*s)
}

// IsSliceNilOrEmpty returns true of the slice is nil or has 0 length (empty).
func IsSliceNilOrEmpty[T any](s []T) bool {
	return s == nil || len(s) == 0
}
