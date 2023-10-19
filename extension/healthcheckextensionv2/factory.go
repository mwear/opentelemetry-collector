// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package healthcheckextensionv2 // import "go.opentelemetry.io/collector/extension/healthcheckextensionv2"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/extension"

	"go.opentelemetry.io/collector/extension/healthcheckextensionv2/internal/metadata"
)

const (
	// Use 0.0.0.0 to make the health check endpoint accessible
	// in container orchestration environments like Kubernetes.
	defaultEndpoint = "0.0.0.0:13133"
)

// NewFactory creates a factory for HealthCheck extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		metadata.ExtensionStability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		HTTPServerSettings: confighttp.HTTPServerSettings{
			Endpoint: defaultEndpoint,
		},
		Path:    "/",
		Verbose: true,
	}
}

func createExtension(_ context.Context, set extension.CreateSettings, cfg component.Config) (extension.Extension, error) {
	config := cfg.(*Config)

	return new(*config, set.TelemetrySettings), nil
}
