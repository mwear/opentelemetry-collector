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
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/translator/internaldata"
)

type testCase struct {
	name string
	inMN [][]*metricspb.Metric // input Metric batches
}

var (
	standardTests = []testCase{
		{
			name: "test",
			inMN: [][]*metricspb.Metric{metricsWithName([]string{"foo"})},
		},
	}
)

func TestPrometheusDiscoveryProcessor(t *testing.T) {
	for _, test := range standardTests {
		t.Run(test.name, func(t *testing.T) {
			// next stores the results of the filter metric processor
			next := new(consumertest.MetricsSink)
			cfg := &Config{
				ProcessorSettings: config.NewProcessorSettings(typeStr),
			}
			factory := NewFactory()
			pdp, err := factory.CreateMetricsProcessor(
				context.Background(),
				component.ProcessorCreateParams{
					Logger: zap.NewNop(),
				},
				cfg,
				next,
			)
			assert.NotNil(t, pdp)
			assert.Nil(t, err)

			caps := pdp.GetCapabilities()
			assert.True(t, caps.MutatesConsumedData)
			ctx := context.Background()
			assert.NoError(t, pdp.Start(ctx, nil))

			mds := make([]internaldata.MetricsData, len(test.inMN))
			for i, metrics := range test.inMN {
				mds[i] = internaldata.MetricsData{
					Metrics: metrics,
				}
			}
			cErr := pdp.ConsumeMetrics(context.Background(), internaldata.OCSliceToMetrics(mds))
			assert.Nil(t, cErr)

			assert.NoError(t, pdp.Shutdown(ctx))
		})
	}
}

func metricsWithName(names []string) []*metricspb.Metric {
	ret := make([]*metricspb.Metric, len(names))
	now := time.Now()
	for i, name := range names {
		ret[i] = &metricspb.Metric{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name: name,
				Type: metricspb.MetricDescriptor_GAUGE_INT64,
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					Points: []*metricspb.Point{
						{
							Timestamp: timestamppb.New(now.Add(10 * time.Second)),
							Value: &metricspb.Point_Int64Value{
								Int64Value: int64(123),
							},
						},
					},
				},
			},
		}
	}
	return ret
}
