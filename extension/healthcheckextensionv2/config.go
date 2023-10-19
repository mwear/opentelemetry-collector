// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package healthcheckextensionv2 // import "go.opentelemetry.io/collector/extension/healthcheckextensionv2"

import "go.opentelemetry.io/collector/config/confighttp"

// Config has the configuration for the extension enabling the health check
// extension, used to report the health status of the service.
type Config struct {
	confighttp.HTTPServerSettings `mapstructure:",squash"`

	// Path represents the path the health check service will serve.
	// The default path is "/".
	Path string `mapstructure:"path"`

	Verbose bool `mapstructure:"verbose"`
}
