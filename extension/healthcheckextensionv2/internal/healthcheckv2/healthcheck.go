// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package healthcheckv2 // import "go.opentelemetry.io/collector/extension/healthcheckextensionv2/internal/healthcheckv2"

import (
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/healthcheckextensionv2/internal/status"
)

type HealthCheck struct {
	aggregator    *status.Aggregator
	responseCodes map[component.Status]int
}

func (hc *HealthCheck) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := hc.aggregator.Current()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(hc.responseCodes[current.Status()])

		body, _ := json.Marshal(current)
		_, _ = w.Write(body)
	})
}

func (hc *HealthCheck) StatusChanged(source *component.InstanceID, event *component.StatusEvent) {
	hc.aggregator.StatusChanged(source, event)
}

func New() *HealthCheck {
	return &HealthCheck{
		aggregator: status.NewAggregator(),
		responseCodes: map[component.Status]int{
			component.StatusNone:             http.StatusServiceUnavailable,
			component.StatusStarting:         http.StatusServiceUnavailable,
			component.StatusOK:               http.StatusOK,
			component.StatusRecoverableError: http.StatusServiceUnavailable,
			component.StatusPermanentError:   http.StatusBadRequest,
			component.StatusFatalError:       http.StatusInternalServerError,
			component.StatusStopping:         http.StatusServiceUnavailable,
			component.StatusStopped:          http.StatusServiceUnavailable,
		},
	}
}
