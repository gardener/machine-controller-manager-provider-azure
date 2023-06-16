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

// IsNilAndEmptyStringPtr returns true if the string pointer is nil or the return value of IsEmptyString(s).
func IsNilAndEmptyStringPtr(s *string) bool {
	if s == nil {
		return true
	}
	return IsEmptyString(*s)
}
