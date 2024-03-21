// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testhelp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

// Error codes
const (
	// ErrorCodeResourceNotFound is the error code returned in Azure response header when resource is not found.
	ErrorCodeResourceNotFound = "ResourceNotFound"
	// ErrorCodeResourceGroupNotFound is the error code returned in Azure response header when resource group is not found.
	ErrorCodeResourceGroupNotFound = "ResourceGroupNotFound"
	// ErrorCodeOperationNotAllowed is the error code returned in Azure response if an operation is not allowed on a resource.
	ErrorCodeOperationNotAllowed = "OperationNotAllowed"
	// ErrorCodeBadRequest is the error code returned in Azure response if the request is not as per the spec. In some cases this code is returned in place of not found.
	ErrorCodeBadRequest = "BadRequest"
	// ErrorCodeVMImageNotFound is the error code returned in Azure response if the VM Image is not found.
	// We have created 2 different constants for not found error because Azure is not consistent. For certain resources it returns
	// ResourceNotFound while for others it just returns NotFound.
	ErrorCodeVMImageNotFound = "NotFound"
	// ErrorCodeSubnetNotFound is the error code returned in Azure response if the subnet is not found.
	// We have created 2 different constants for not found error because Azure is not consistent. For certain resources it returns
	// ResourceNotFound while for others it just returns NotFound.
	ErrorCodeSubnetNotFound = "NotFound"
	// ErrorCodeReferencedResourceNotFound is the error code returned in Azure response if a referenced resource
	// is not found to exist.
	ErrorCodeReferencedResourceNotFound = "NotFound"
	// ErrorCodeAttachDiskWhileBeingDetached is the error code returned in Azure response if there is an attempt to update the DeleteOptions for
	// associated Disks when the Disk is currently getting detached.
	ErrorCodeAttachDiskWhileBeingDetached = "AttachDiskWhileBeingDetached"
)

// ContextTimeoutError creates an error mimicking timeout of a context.
func ContextTimeoutError(parentCtx context.Context, timeout time.Duration) error {
	opCtx, cancelFn := context.WithTimeout(parentCtx, timeout)
	defer cancelFn()
	select {
	case <-opCtx.Done():
		return opCtx.Err()
	}
}

// ResourceNotFoundErr creates a resource not found error setting azure specific error code as a response header.
func ResourceNotFoundErr(errorCode string) error {
	headers := http.Header{}
	headers.Set("x-ms-error-code", errorCode)
	resp := &http.Response{
		Status:     "404 NotFound",
		StatusCode: 404,
		Header:     headers,
	}
	return runtime.NewResponseError(resp)
}

// ConfiguredRelatedResourceNotFound creates a resource not found error specifically for cases where
// the spec used to create a resource refers to another related attached resource which does not exist.
// Example: If during VM creation, a NIC which has a reference in armcompute.VirtualMachine does not exist
// then this function should be called.
func ConfiguredRelatedResourceNotFound(errorCode string, referencedResourceID string) error {
	headers := http.Header{}
	headers.Set("x-ms-error-code", errorCode)
	resp := &http.Response{
		Status:     "404 NotFound",
		StatusCode: 404,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(fmt.Sprintf("Resource %s not found", referencedResourceID))),
	}
	return runtime.NewResponseError(resp)
}

// ConflictErr creates a conflict error setting azure specific error code as a response header.
func ConflictErr(errorCode string) error {
	headers := http.Header{}
	headers.Set("x-ms-error-code", errorCode)
	resp := &http.Response{
		Status:     "409 Conflict",
		StatusCode: 409,
		Header:     headers,
	}
	return runtime.NewResponseError(resp)
}

// InternalServerError creates an internal server error setting the azure specific error code as response header.
func InternalServerError(errorCode string) error {
	headers := http.Header{}
	headers.Set("x-ms-error-code", errorCode)
	resp := &http.Response{
		Status:     "500 Internal Server Error",
		StatusCode: 500,
		Header:     headers,
	}
	return runtime.NewResponseError(resp)
}

// BadRequestError creates a bad request error setting azure specific error code as a response header.
func BadRequestError(errorCode string) error {
	headers := http.Header{}
	headers.Set("x-ms-error-code", errorCode)
	resp := &http.Response{
		Status:     "400 Bad Request",
		StatusCode: 400,
		Header:     headers,
	}
	return runtime.NewResponseError(resp)
}
