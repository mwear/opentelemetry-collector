// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package healthcheckextensionv2 // import "go.opentelemetry.io/collector/extension/healthcheckextensionv2"

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/healthcheckextensionv2/internal/healthcheckv2"
	"go.uber.org/zap"
)

type healthCheckExtension struct {
	config      Config
	telemetry   component.TelemetrySettings
	healthCheck *healthcheckv2.HealthCheck
	server      *http.Server
	stopCh      chan struct{}
}

func (hc *healthCheckExtension) Start(_ context.Context, host component.Host) error {
	hc.telemetry.Logger.Info("Starting healthcheck extension V2", zap.Any("config", hc.config))

	ln, err := hc.config.ToListener()
	if err != nil {
		return fmt.Errorf("failed to bind to address %s: %w", hc.config.Endpoint, err)
	}

	mux := http.NewServeMux()
	mux.Handle(hc.config.Path, hc.healthCheck.Handler())
	hc.server, err = hc.config.ToServer(host, hc.telemetry, mux)
	if err != nil {
		return err
	}

	go func() {
		defer close(hc.stopCh)
		hc.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusOK))
		if err = hc.server.Serve(ln); !errors.Is(err, http.ErrServerClosed) && err != nil {
			hc.telemetry.ReportComponentStatus(component.NewFatalErrorEvent(err))
		}
	}()

	return nil
}

func (hc *healthCheckExtension) Shutdown(_ context.Context) error {
	if hc.server == nil {
		return nil
	}
	err := hc.server.Close()
	if hc.stopCh != nil {
		<-hc.stopCh
	}
	return err
}

func (hc *healthCheckExtension) ComponentStatusChanged(source *component.InstanceID, event *component.StatusEvent) {
	fmt.Printf("component status changed; source: %v; event: %v\n", source, event)
	hc.healthCheck.StatusChanged(source, event)
}

func new(config Config, settings component.TelemetrySettings) *healthCheckExtension {
	hc := &healthCheckExtension{
		config:      config,
		healthCheck: healthcheckv2.New(),
		telemetry:   settings,
		stopCh:      make(chan struct{}),
	}

	return hc
}
