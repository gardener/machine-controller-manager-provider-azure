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
	// ErrorOperationNotAllowed is the error code returned in Azure response if an operation is not allowed on a resource.
	ErrorOperationNotAllowed = "OperationNotAllowed"
	// ErrorBadRequest is the error code returned in Azure response if the request is not as per the spec. In some cases this code is returned in place of not found.
	ErrorBadRequest = "BadRequest"
	// ErrorCodeVMImageNotFound is the error code returned in Azure response if the VM Image is not found.
	// We have created 2 different constants for not found error because Azure is not consistent. For certain resources it returns
	// ResourceNotFound while for others it just returns NotFound.
	ErrorCodeVMImageNotFound = "NotFound"
	// ErrorCodeSubnetNotFound is the error code returned in Azure response if the subnet is not found.
	// We have created 2 different constants for not found error because Azure is not consistent. For certain resources it returns
	// ResourceNotFound while for others it just returns NotFound.
	ErrorCodeSubnetNotFound = "NotFound"
	// ErrorCodeReferencedResourceNotFound	 is the error code returned in Azure response if a referenced resource
	// is not found to exist.
	ErrorCodeReferencedResourceNotFound = "NotFound"
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
