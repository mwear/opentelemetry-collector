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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"path"
	"testing"
	"time"

	"go.uber.org/zap"
)

var logger = zap.NewNop()

func TestEndToEnd(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config_discovery.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	config := cfg.Receivers["prometheus/discovery"].(*Config)

	cms := new(consumertest.MetricsSink)
	r := newPrometheusReceiver(logger, config, cms)

	err = r.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// default sync time of the discovery channel is 5 seconds.
	time.Sleep(6 * time.Second)

	dataPoints := cms.AllMetrics()[0].ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntGauge().DataPoints()
	dataPoints.At(0)
	// TODO: verify these dataPoints, probably need to fix the formatter.
}
