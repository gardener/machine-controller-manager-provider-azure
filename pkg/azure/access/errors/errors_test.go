// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	. "github.com/onsi/gomega"
)

type input struct {
	inputError   error
	expectedCode codes.Code
}

// exceedingQuotaWrapper wraps an *azcore.ResponseError but returns an
// Error() string that contains 'exceeding quota' so regex matching works
// in GetMatchingErrorCode while still allowing errors.As to find the
// underlying *azcore.ResponseError.
type exceedingQuotaWrapper struct {
	resp *azcore.ResponseError
}

func (w *exceedingQuotaWrapper) Error() string { return "exceeding quota: reached subscription limit" }
func (w *exceedingQuotaWrapper) Unwrap() error { return w.resp }

// falseExceedingQuotaWrapper wraps an *azcore.ResponseError but returns an
// Error() string that does not contain 'exceeding quota' so regex matching fails
// in GetMatchingErrorCode while still allowing errors.As to find the
// underlying *azcore.ResponseError.
type falseExceedingQuotaWrapper struct {
	resp *azcore.ResponseError
}

func (w *falseExceedingQuotaWrapper) Error() string {
	return "operation not allowed: some other reason"
}
func (w *falseExceedingQuotaWrapper) Unwrap() error { return w.resp }

// notSupportedWrapper wraps an *azcore.ResponseError but returns an
// Error() string that contains 'not support' so regex matching works
// in GetMatchingErrorCode while still allowing errors.As to find the
// underlying *azcore.ResponseError.
type notSupportedWrapper struct {
	resp *azcore.ResponseError
}

func (w *notSupportedWrapper) Error() string {
	return "not supported: operation is not supported on this resource"
}
func (w *notSupportedWrapper) Unwrap() error { return w.resp }

// falseNotSupportedWrapper wraps an *azcore.ResponseError but returns an
// Error() string that does not contain 'not support' so regex matching fails
// in GetMatchingErrorCode while still allowing errors.As to find the
// underlying *azcore.ResponseError.
type falseNotSupportedWrapper struct {
	resp *azcore.ResponseError
}

func (w *falseNotSupportedWrapper) Error() string { return "bad request: some other reason" }
func (w *falseNotSupportedWrapper) Unwrap() error { return w.resp }

func TestGetMCMErrorCodeForCreateMachine(t *testing.T) {
	table := []input{
		{inputError: &azcore.ResponseError{ErrorCode: ZonalAllocationFailedAzErrorCode}, expectedCode: codes.ResourceExhausted},
		{inputError: &azcore.ResponseError{ErrorCode: SkuNotAvailableAzErrorCode}, expectedCode: codes.ResourceExhausted},
		{inputError: &azcore.ResponseError{ErrorCode: AllocationFailedAzErrorCode}, expectedCode: codes.ResourceExhausted},
		{inputError: &azcore.ResponseError{ErrorCode: ResourceQuotaExceededAzErrorCode}, expectedCode: codes.ResourceExhausted},
		{inputError: &exceedingQuotaWrapper{resp: &azcore.ResponseError{ErrorCode: OperationNotAllowedAzErrorCode}}, expectedCode: codes.ResourceExhausted},
		{inputError: &notSupportedWrapper{resp: &azcore.ResponseError{ErrorCode: BadRequestAzErrorCode}}, expectedCode: codes.ResourceExhausted},
		{inputError: &falseExceedingQuotaWrapper{resp: &azcore.ResponseError{ErrorCode: OperationNotAllowedAzErrorCode}}, expectedCode: codes.Internal},
		{inputError: &falseNotSupportedWrapper{&azcore.ResponseError{ErrorCode: BadRequestAzErrorCode}}, expectedCode: codes.Internal},
		{inputError: &azcore.ResponseError{ErrorCode: "unknown error"}, expectedCode: codes.Internal},
	}
	g := NewWithT(t)
	for _, entry := range table {
		g.Expect(GetMatchingErrorCode(entry.inputError)).To(Equal(entry.expectedCode), "for input error: %v", entry.inputError)
	}
}
