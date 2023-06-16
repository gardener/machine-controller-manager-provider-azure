package instrument

import (
	"time"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusProviderLabelKey   = "provider"
	prometheusProviderLabelValue = "azure"
	prometheusServiceLabelKey    = "service"
)

// RecordAzAPIMetric records a prometheus metric for Azure API calls.
// * If there is an error then it will increment the APIFailedRequestCount counter vec metric.
// * If the Azure API call is successful then it will record 2 metrics:
//   - It will increment APIRequestCount counter vec metric. (Ideally this should have been named as APISuccessRequestCount)
//   - It will compute the time taken for API call completion and record it.
//
// NOTE: If this function is called via `defer` then please keep in mind that parameters passed to defer are evaluated at the time of definition.
// So if you have an error that is computed later in the function then ensure that you use named return parameters.
func RecordAzAPIMetric(err error, azServiceName string, invocationTime time.Time) {
	if err != nil {
		metrics.APIFailedRequestCount.With(
			prometheus.Labels{
				prometheusProviderLabelKey: prometheusProviderLabelValue,
				prometheusServiceLabelKey:  azServiceName,
			},
		).Inc()
		return
	}

	// No error, record metrics for successful API call.
	metrics.APIRequestCount.With(
		prometheus.Labels{
			prometheusProviderLabelKey: prometheusProviderLabelValue,
			prometheusServiceLabelKey:  azServiceName,
		},
	)
	// compute the time taken to complete the AZ service call
	//elapsed := time.Since(invocationTime)
	// introduce a new metric in MCM provider that will capture API call duration. Once that is
	// introduced then uncomment the above line and use it as a value for that metric.
}
