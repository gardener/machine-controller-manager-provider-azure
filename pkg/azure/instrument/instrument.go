package instrument

import (
	"time"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
)

const prometheusProviderLabelValue = "azure"

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
		metrics.APIFailedRequestCount.
			WithLabelValues(prometheusProviderLabelValue, azServiceName).
			Inc()
		return
	}

	// No error, record metrics for successful API call.
	metrics.APIRequestCount.
		WithLabelValues(
			prometheusProviderLabelValue,
			azServiceName,
		).Inc()

	// compute the time taken to complete the AZ service call and record it as a metric
	elapsed := time.Since(invocationTime)
	metrics.APIRequestDuration.WithLabelValues(
		prometheusProviderLabelValue,
		azServiceName,
	).Observe(elapsed.Seconds())
}

// RecordDriverAPIMetric records a prometheus metric capturing the total duration of a successful execution for
// any driver method (e.g. CreateMachine, DeleteMachine etc.).
func RecordDriverAPIMetric(err error, operation string, invocationTime time.Time) {
	if err != nil {
		// currently we only record duration for successful completion of driver methods
		return
	}
	// compute the time taken to complete the AZ service call and record it as a metric
	elapsed := time.Since(invocationTime)
	metrics.DriverAPIRequestDuration.WithLabelValues(
		prometheusProviderLabelValue,
		operation,
	).Observe(elapsed.Seconds())
}
