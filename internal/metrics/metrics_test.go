package metrics

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func TestInit(t *testing.T) {
	// Init with noop meter (no provider registered) should not panic
	Init()

	if ToolCallsTotal == nil {
		t.Fatal("ToolCallsTotal should not be nil after Init")
	}
	if ToolRouteDuration == nil {
		t.Fatal("ToolRouteDuration should not be nil after Init")
	}
	if RequestsTotal == nil {
		t.Fatal("RequestsTotal should not be nil after Init")
	}
	if ActiveSessions == nil {
		t.Fatal("ActiveSessions should not be nil after Init")
	}
	if ServerHealth == nil {
		t.Fatal("ServerHealth should not be nil after Init")
	}
	if ToolListTotal == nil {
		t.Fatal("ToolListTotal should not be nil after Init")
	}
	if KialiExtRequestsTotal == nil {
		t.Fatal("KialiExtRequestsTotal should not be nil after Init")
	}
	if KialiExtResponseTime == nil {
		t.Fatal("KialiExtResponseTime should not be nil after Init")
	}
}

func TestCountersDoNotPanic(t *testing.T) {
	Init()
	ctx := context.Background()
	attrs := metric.WithAttributes(
		attribute.String("tool_name", "test_tool"),
		attribute.String("mcp_server_name", "test_server"),
	)

	// these should not panic even with noop meter
	ToolCallsTotal.Add(ctx, 1, attrs)
	ToolRouteDuration.Record(ctx, 0.5, attrs)
	RequestsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", "tools/call"),
		attribute.String("component", "router"),
	))
	ActiveSessions.Add(ctx, 1)
	ActiveSessions.Add(ctx, -1)
	ServerHealth.Add(ctx, 1, metric.WithAttributes(
		attribute.String("server_name", "test"),
	))
	ToolListTotal.Add(ctx, 1)

	// kiali extension metrics should not panic
	kialiAttrs := NewKialiGatewayToUpstream("mcp-test/weather", "200")
	KialiExtRequestsTotal.Add(ctx, 1, kialiAttrs.Attributes())
	KialiExtResponseTime.Record(ctx, 0.1, kialiAttrs.Attributes())
}

func TestServerNameParts(t *testing.T) {
	tests := []struct {
		input    string
		wantNS   string
		wantName string
	}{
		{"mcp-test/weather-service", "mcp-test", "weather-service"},
		{"default/my-server", "default", "my-server"},
		{"simple-name", "unknown", "simple-name"},
		{"a/b/c", "a", "b/c"},
	}
	for _, tt := range tests {
		ns, name := ServerNameParts(tt.input)
		if ns != tt.wantNS || name != tt.wantName {
			t.Errorf("ServerNameParts(%q) = (%q, %q), want (%q, %q)", tt.input, ns, name, tt.wantNS, tt.wantName)
		}
	}
}

func TestKialiExtAttrsLabels(t *testing.T) {
	attrs := NewKialiGatewayToUpstream("mcp-test/weather", "200")
	if attrs.Extension != "mcp-gateway" {
		t.Errorf("expected extension=mcp-gateway, got %s", attrs.Extension)
	}
	if attrs.DestNS != "mcp-test" {
		t.Errorf("expected dest_namespace=mcp-test, got %s", attrs.DestNS)
	}
	if attrs.DestName != "weather" {
		t.Errorf("expected dest_name=weather, got %s", attrs.DestName)
	}
	if attrs.SourceName != "mcp-gateway-istio" {
		t.Errorf("expected source_name=mcp-gateway-istio, got %s", attrs.SourceName)
	}
	if attrs.SourceNS != "gateway-system" {
		t.Errorf("expected source_namespace=gateway-system, got %s", attrs.SourceNS)
	}
	if attrs.SourceCluster != "Kubernetes" {
		t.Errorf("expected source_cluster=Kubernetes, got %s", attrs.SourceCluster)
	}
	if attrs.SourceIsRoot != "true" {
		t.Errorf("expected source_is_root=true, got %s", attrs.SourceIsRoot)
	}
	if attrs.Protocol != "http" {
		t.Errorf("expected protocol=http, got %s", attrs.Protocol)
	}

	// verify Attributes() doesn't panic
	opt := attrs.Attributes()
	if opt == nil {
		t.Fatal("Attributes() returned nil")
	}
}
