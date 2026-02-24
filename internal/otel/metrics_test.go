package otel

import (
	"context"
	"testing"
)

func TestNewMetricsProvider_disabled(t *testing.T) {
	cfg := &Config{
		Endpoint:    "",
		ServiceName: "test",
	}

	_, err := NewMetricsProvider(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when endpoint is empty")
	}
}

func TestNewMetricsProvider_invalidScheme(t *testing.T) {
	cfg := &Config{
		Endpoint:    "ftp://collector:4318",
		ServiceName: "test",
	}

	// MetricsEndpoint falls back to base when no override set
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "ftp://collector:4318")

	_, err := NewMetricsProvider(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestNewMetricExporter_http(t *testing.T) {
	exp, err := newMetricExporter(context.Background(), "http://localhost:4318", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
}

func TestNewMetricExporter_grpc(t *testing.T) {
	exp, err := newMetricExporter(context.Background(), "rpc://localhost:4317", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
}
