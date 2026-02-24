// Package otel provides OpenTelemetry tracing and logging integration
package otel

import "k8s.io/utils/env"

// Config holds configuration for OpenTelemetry
type Config struct {
	Endpoint       string
	Insecure       bool
	ServiceName    string
	ServiceVersion string
	GitSHA         string
	GitDirty       string
}

// NewConfig creates OTel configuration from environment variables
func NewConfig(gitSHA, dirty, version string) *Config {
	endpoint := env.GetString("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	insecure, _ := env.GetBool("OTEL_EXPORTER_OTLP_INSECURE", false)

	serviceName := env.GetString("OTEL_SERVICE_NAME", "mcp-gateway")
	serviceVersion := env.GetString("OTEL_SERVICE_VERSION", version)

	return &Config{
		Endpoint:       endpoint,
		Insecure:       insecure,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		GitSHA:         gitSHA,
		GitDirty:       dirty,
	}
}

// TracesEndpoint returns the endpoint for traces, with signal-specific override support
func (c *Config) TracesEndpoint() string {
	if endpoint := env.GetString("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""); endpoint != "" {
		return endpoint
	}
	return c.Endpoint
}

// LogsEndpoint returns the endpoint for logs, with signal-specific override support
func (c *Config) LogsEndpoint() string {
	if endpoint := env.GetString("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", ""); endpoint != "" {
		return endpoint
	}
	return c.Endpoint
}

// TracesEnabled returns true if tracing is enabled
func (c *Config) TracesEnabled() bool {
	return c.TracesEndpoint() != ""
}

// LogsEnabled returns true if logs export is enabled
func (c *Config) LogsEnabled() bool {
	return c.LogsEndpoint() != ""
}

// MetricsEndpoint returns the endpoint for metrics, with signal-specific override support
func (c *Config) MetricsEndpoint() string {
	if endpoint := env.GetString("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", ""); endpoint != "" {
		return endpoint
	}
	return c.Endpoint
}

// MetricsEnabled returns true if metrics export is enabled
func (c *Config) MetricsEnabled() bool {
	return c.MetricsEndpoint() != ""
}

// Enabled returns true if any telemetry signal is enabled
func (c *Config) Enabled() bool {
	return c.TracesEnabled() || c.LogsEnabled() || c.MetricsEnabled()
}
