package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

const (
	ResourceQuotaExceededAzErrorCode = "ResourceQuotaExceeded"
	//OverConstrainedAllocationRequest
	CorrelationRequestIDAzHeaderKey = "x-ms-correlation-request-id"
	RequestIDAzHeaderKey            = "x-ms-request-id"
	ErrorCodeAzHeaderKey            = "x-ms-error-code"
	ClientRequestIDAzHeaderKey      = "x-ms-client-request-id"
)

var (
	// Raised https://github.com/Azure/azure-sdk-for-go/issues/21094 to prevent hard coding these here and instead
	// use well-maintained constants defined in the Azure SDK.
	lookupResponseHeaderKeys = sets.New(
		CorrelationRequestIDAzHeaderKey,
		RequestIDAzHeaderKey,
		ErrorCodeAzHeaderKey,
		ClientRequestIDAzHeaderKey,
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

// GetMatchingErrorCode gets a matching codes.Code for the given azure error code.
func GetMatchingErrorCode(err error) codes.Code {
	azHeaders := traceResponseHeaders(err)
	azErrorCode, ok := azHeaders[ErrorCodeAzHeaderKey]
	if ok {
		switch azErrorCode {
		case ResourceQuotaExceededAzErrorCode:
			return codes.ResourceExhausted
		default:
			return codes.Internal
		}
	}
	return codes.Internal
}
