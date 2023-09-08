// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicetelemetry // import "go.opentelemetry.io/collector/service/internal/servicetelemetry"

import (
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// NewNopSettings returns a new nop settings for Create* functions.
func NewNopSettings() Settings {
	return Settings{
		Logger:                zap.NewNop(),
		TracerProvider:        trace.NewNoopTracerProvider(),
		MeterProvider:         noop.NewMeterProvider(),
		MetricsLevel:          configtelemetry.LevelNone,
		Resource:              pcommon.NewResource(),
		ReportComponentStatus: func(*component.InstanceID, component.Status, ...component.StatusEventOption) {},
	}
}
