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
}
