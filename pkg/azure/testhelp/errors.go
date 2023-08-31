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
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

// Error codes
const (
	ErrorCodeResourceNotFound      = "ResourceNotFound"
	ErrorCodeResourceGroupNotFound = "ResourceGroupNotFound"
	ErrorCodePatchResourceNotFound = "PatchResourceNotFound"
	ErrorOperationNotAllowed       = "OperationNotAllowed"
	ErrorBadRequest                = "BadRequest"
	ErrorCodeVMImageNotFound       = "NotFound"
	ErrorCodeSubnetNotFound        = "NotFound"
)

func ContextTimeoutError(parentCtx context.Context, timeout time.Duration) error {
	opCtx, cancelFn := context.WithTimeout(parentCtx, timeout)
	defer cancelFn()
	select {
	case <-opCtx.Done():
		return opCtx.Err()
	}
}

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
