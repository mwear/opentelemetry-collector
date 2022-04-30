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

package service // import "go.opentelemetry.io/collector/service"

import (
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/status"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/internal/builder"
	"go.opentelemetry.io/collector/service/internal/extensions"
)

var _ component.Host = (*serviceHost)(nil)

type serviceHost struct {
	asyncErrorChannel   chan error
	factories           component.Factories
	telemetry           component.TelemetrySettings
	statusNotifications *status.Notifications

	builtExporters  builder.Exporters
	builtReceivers  builder.Receivers
	builtPipelines  builder.BuiltPipelines
	builtExtensions extensions.Extensions
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (host *serviceHost) ReportFatalError(err error) {
	host.asyncErrorChannel <- err
}

func (host *serviceHost) GetFactory(kind component.Kind, componentType config.Type) component.Factory {
	switch kind {
	case component.KindReceiver:
		return host.factories.Receivers[componentType]
	case component.KindProcessor:
		return host.factories.Processors[componentType]
	case component.KindExporter:
		return host.factories.Exporters[componentType]
	case component.KindExtension:
		return host.factories.Extensions[componentType]
	}
	return nil
}

func (host *serviceHost) GetExtensions() map[config.ComponentID]component.Extension {
	return host.builtExtensions.ToMap()
}

func (host *serviceHost) GetExporters() map[config.DataType]map[config.ComponentID]component.Exporter {
	return host.builtExporters.ToMapByDataType()
}

func (host *serviceHost) RegisterStatusListener(options ...status.ListenerOption) status.UnregisterFunc {
	return host.statusNotifications.RegisterListener(options...)
}

// ReportStatus is an implementation of Host.ReportStatus. Note, that reporting a status.EventError
// with an error wrapped by componenterror.NewFatal() will cause the collector process to terminate
// with a non-zero exit code.
func (host *serviceHost) ReportStatus(eventType status.EventType, componentID config.ComponentID, options ...status.StatusEventOption) {
	event := status.NewStatusEvent(eventType, componentID, options...)
	if err := host.statusNotifications.ReportStatus(event); err != nil {
		host.telemetry.Logger.Warn("Service failed to report status", zap.Error(err))
	}
	if eventType == status.EventError && event.Error != nil && componenterror.IsFatal(event.Error) {
		host.asyncErrorChannel <- event.Error
	}
}
