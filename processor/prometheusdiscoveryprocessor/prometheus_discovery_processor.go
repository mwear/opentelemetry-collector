// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusdiscoveryprocessor

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type prometheusDiscoveryProcessor struct {
	cfg    *Config
	logger *zap.Logger
	//attributeCache map[string]map[string]pdata.StringMap
}

func newPrometheusDiscoveryProcessor(logger *zap.Logger, cfg *Config) (*prometheusDiscoveryProcessor, error) {

	logger.Info("Prometheus Discovery Processor configured")

	return &prometheusDiscoveryProcessor{
		cfg:    cfg,
		logger: logger,
	}, nil
}

// ProcessMetrics looks for "up" metrics and when found will store attributes collected from service
// discovery. Non "up" metrics with matching job and instance attributes will be enriched with the
// discovered attributes.
func (pdp *prometheusDiscoveryProcessor) ProcessMetrics(_ context.Context, pdm pdata.Metrics) (pdata.Metrics, error) {
	rms := pdm.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ms := ilms.At(j).Metrics()
			for k := 0; k < ms.Len(); k++ {
				//met := ms.At(k)

				/*
					if met is an "up" metric
					  update the attribute cache with the discovered labels
					else if metric has a job and instance attribute and the cache has a matching entry, e.g., attributeCache[job][instance]
					  merge attributes from cache with attributes on the metric point
				*/
			}
		}
	}

	return pdm, nil
}
