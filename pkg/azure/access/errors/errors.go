package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

var (
	// Raised https://github.com/Azure/azure-sdk-for-go/issues/21094 to prevent hard coding these here and instead
	// use well-maintained constants defined in the Azure SDK.
	lookupResponseHeaderKeys = sets.New(
		"x-ms-correlation-request-id",
		"x-ms-request-id",
		"x-ms-error-code",
		"x-ms-client-request-id",
	)
)

// IsNotFoundAzAPIError checks if error is an AZ API error and if it is a 404 response code.
func IsNotFoundAzAPIError(err error) bool {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == http.StatusNotFound
	}
	return false
}

// LogAzAPIError collects additional information from AZ response and logs it as part of the error log message.
func LogAzAPIError(err error, format string, v ...any) {
	if err == nil {
		return
	}
	respHeaders := traceResponseHeaders(err)
	errMsg := fmt.Sprintf(format, v...)
	if len(respHeaders) == 0 {
		klog.Errorf("%s: %+v\n", errMsg, err)
		return
	}
	klog.Errorf("%s : Azure API Response-Headers: %+v Err: %+v\n", errMsg, respHeaders, err)
}

func traceResponseHeaders(err error) map[string]string {
	var respErr *azcore.ResponseError
	headers := make(map[string]string)
	if errors.As(err, &respErr) {
		respHeader := respErr.RawResponse.Header
		for headerKey := range lookupResponseHeaderKeys {
			headerValue := respHeader.Get(headerKey)
			if !utils.IsEmptyString(headerValue) {
				headers[headerKey] = headerValue
			}
		}
	}
	return headers
}
