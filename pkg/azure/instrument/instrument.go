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

package instrument

import (
	"errors"
	"strconv"
	"time"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
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
// any driver method (e.g. CreateMachine, DeleteMachine etc.). In case an error is returned then a failed counter
// metric is recorded.
func RecordDriverAPIMetric(err error, operation string, invocationTime time.Time) {
	if err != nil {
		var (
			statusErr *status.Status
			labels    = []string{prometheusProviderLabelValue, operation}
		)
		if errors.As(err, &statusErr) {
			labels = append(labels, strconv.Itoa(int(statusErr.Code())))
		}
		metrics.DriverFailedAPIRequests.
			WithLabelValues(labels...).
			Inc()
		return
	}
	// compute the time taken to complete the AZ service call and record it as a metric
	elapsed := time.Since(invocationTime)
	metrics.DriverAPIRequestDuration.WithLabelValues(
		prometheusProviderLabelValue,
		operation,
	).Observe(elapsed.Seconds())
}
