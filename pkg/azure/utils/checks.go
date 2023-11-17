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
