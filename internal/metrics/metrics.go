// Package metrics defines application-level MCP metrics exported via OpenTelemetry
package metrics

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "mcp-gateway"

var (
	// ToolCallsTotal counts tool call requests
	ToolCallsTotal metric.Int64Counter

	// ToolCallDuration records tool call latency in seconds
	ToolCallDuration metric.Float64Histogram

	// RequestsTotal counts all MCP requests by method and component
	RequestsTotal metric.Int64Counter

	// ActiveSessions tracks currently active client sessions
	ActiveSessions metric.Int64UpDownCounter

	// ServerHealth tracks upstream server health (1=healthy, -1 on transition to unhealthy)
	ServerHealth metric.Int64UpDownCounter

	// ToolListTotal counts tools/list requests
	ToolListTotal metric.Int64Counter
)

// Init initializes all metric instruments from the global MeterProvider.
// Safe to call before a MeterProvider is registered; instruments will use a noop meter
// and begin recording once a real provider is set.
func Init() {
	meter := otel.Meter(meterName)

	ToolCallsTotal, _ = meter.Int64Counter("mcp.tool_calls_total",
		metric.WithDescription("Total number of MCP tool call requests"),
	)

	ToolCallDuration, _ = meter.Float64Histogram("mcp.tool_call_duration_seconds",
		metric.WithDescription("Duration of MCP tool calls in seconds"),
	)

	RequestsTotal, _ = meter.Int64Counter("mcp.requests_total",
		metric.WithDescription("Total number of MCP requests"),
	)

	ActiveSessions, _ = meter.Int64UpDownCounter("mcp.active_sessions",
		metric.WithDescription("Number of currently active MCP client sessions"),
	)

	ServerHealth, _ = meter.Int64UpDownCounter("mcp.server_health",
		metric.WithDescription("Health status of upstream MCP servers"),
	)

	ToolListTotal, _ = meter.Int64Counter("mcp.tool_list_total",
		metric.WithDescription("Total number of tools/list requests"),
	)
}
