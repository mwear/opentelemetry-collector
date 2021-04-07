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

package prometheusdiscoveryreceiver

import (
	"context"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/prometheusdiscoveryreceiver/internal"
	"go.uber.org/zap"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

const transport = "http"

// pReceiver is the type that provides Prometheus scraper/receiver functionality.
type pReceiver struct {
	cfg        *Config
	consumer   consumer.Metrics
	cancelFunc context.CancelFunc

	logger *zap.Logger
}

// New creates a new prometheus.Receiver reference.
func newPrometheusReceiver(logger *zap.Logger, cfg *Config, next consumer.Metrics) *pReceiver {
	pr := &pReceiver{
		cfg:      cfg,
		consumer: next,
		logger:   logger,
	}
	return pr
}

// Start is the method that starts Prometheus scraping and it
// is controlled by having previously defined a Configuration using perhaps New.
func (r *pReceiver) Start(_ context.Context, host component.Host) error {
	discoveryCtx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel

	logger := internal.NewZapToGokitLogAdapter(r.logger)

	discoveryManager := discovery.NewManager(discoveryCtx, logger)
	discoveryCfg := make(map[string]discovery.Configs)
	for _, scrapeConfig := range r.cfg.PrometheusConfig.ScrapeConfigs {
		discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfigs
	}
	if err := discoveryManager.ApplyConfig(discoveryCfg); err != nil {
		return err
	}
	go func() {
		if err := discoveryManager.Run(); err != nil {
			r.logger.Error("Discovery manager failed", zap.Error(err))
			host.ReportFatalError(err)
		}
	}()

	go r.generatePresent(discoveryManager.SyncCh())

	return nil
}

func (r *pReceiver) generatePresent(syncCh <-chan map[string][]*targetgroup.Group) {
	for {
		select {
		case tgs := <-syncCh:
			r.formatGroups(tgs)
		}
	}
}

// Shutdown stops and cancels the underlying Prometheus scrapers.
func (r *pReceiver) Shutdown(context.Context) error {
	r.cancelFunc()
	return nil
}

func (r *pReceiver) formatGroups(tgs map[string][]*targetgroup.Group) {
	metric := pdata.NewMetric()
	metric.SetDataType(pdata.MetricDataTypeIntGauge)
	metric.SetName("present")
	metric.SetDescription("Clear description of what present means")

	instMetrics := pdata.NewInstrumentationLibraryMetrics()
	instMetrics.Metrics().Append(metric)

	for _, groups := range tgs {
		for _, group := range groups {
			for _, target := range group.Targets {
				intDataPoint := pdata.NewIntDataPoint()
				intDataPoint.SetValue(1)
				intDataPoint.SetTimestamp(pdata.TimestampFromTime(time.Now()))
				labels := intDataPoint.LabelsMap()

				for name, value := range group.Labels {
					//TODO: should we validate name and value?
					//TODO: find a way to create this labels once, for a group.
					labels.Insert(string(name), string(value))
				}

				labels.Insert("job", group.Source) //TODO: validate if this is correct.
				labels.Insert("instance", string(target["__address__"]))

				metric.IntGauge().DataPoints().Append(intDataPoint)
			}
		}
	}

	rscMetrics := pdata.NewResourceMetrics()
	rscMetrics.InstrumentationLibraryMetrics().Append(instMetrics)

	ms := pdata.NewMetrics()
	ms.ResourceMetrics().Append(rscMetrics)

	// do some error handling here.
	// TODO: should we use context.Background?
	_ = r.consumer.ConsumeMetrics(context.Background(), ms)

}
