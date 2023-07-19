package fakes

import (
	"context"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
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
