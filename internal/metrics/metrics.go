// Package metrics defines application-level MCP metrics exported via OpenTelemetry
package metrics

import (
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "mcp-gateway"

// kiali extension metric histogram buckets per the kiali extension spec
var kialiResponseTimeBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

var (
	// ToolCallsTotal counts tool call requests
	ToolCallsTotal metric.Int64Counter

	// ToolRouteDuration records router decision latency for tool calls in seconds
	ToolRouteDuration metric.Float64Histogram

	// RequestsTotal counts all MCP requests by method and component
	RequestsTotal metric.Int64Counter

	// ActiveSessions tracks currently active client sessions
	ActiveSessions metric.Int64UpDownCounter

	// ServerHealth tracks upstream server health (1=healthy, -1 on transition to unhealthy)
	ServerHealth metric.Int64UpDownCounter

	// ToolListTotal counts tools/list requests
	ToolListTotal metric.Int64Counter

	// KialiExtRequestsTotal is a kiali extension counter for requests per source/dest edge
	KialiExtRequestsTotal metric.Int64Counter

	// KialiExtResponseTime is a kiali extension histogram for response latency per edge
	KialiExtResponseTime metric.Float64Histogram
)

// KialiExtAttrs holds the label set for a single kiali extension metric recording
type KialiExtAttrs struct {
	Extension     string
	SourceCluster string
	SourceNS      string
	SourceName    string
	SourceIsRoot  string
	Reporter      string
	ReporterID    string
	DestCluster   string
	DestNS        string
	DestName      string
	Protocol      string
	StatusCode    string
	Flags         string
	Secure        string
}

// Attributes returns the label set as metric.MeasurementOption
func (k KialiExtAttrs) Attributes() metric.MeasurementOption {
	return metric.WithAttributeSet(kialiAttrSet(k))
}

// defaultCluster returns the cluster identifier from KIALI_CLUSTER env or a fallback.
// defaults to "Kubernetes" to match Istio/Kiali's default cluster name.
func defaultCluster() string {
	if v := os.Getenv("KIALI_CLUSTER"); v != "" {
		return v
	}
	return "Kubernetes"
}

// defaultReporterID returns a stable reporter identifier from the pod name
func defaultReporterID() string {
	if v, err := os.Hostname(); err == nil {
		return v
	}
	return "mcp-gateway"
}

// defaultSourceName returns the source workload name for kiali extension metrics.
// defaults to the istio gateway deployment name convention (<gateway-name>-istio).
func defaultSourceName() string {
	if v := os.Getenv("KIALI_EXT_SOURCE_NAME"); v != "" {
		return v
	}
	return "mcp-gateway-istio"
}

// defaultSourceNamespace returns the source namespace for kiali extension metrics.
func defaultSourceNamespace() string {
	if v := os.Getenv("KIALI_EXT_SOURCE_NAMESPACE"); v != "" {
		return v
	}
	return "gateway-system"
}

// ServerNameParts splits a server name in "namespace/name" format into its parts.
// If the name has no slash, namespace defaults to "unknown".
func ServerNameParts(serverName string) (namespace, name string) {
	if i := strings.Index(serverName, "/"); i >= 0 {
		return serverName[:i], serverName[i+1:]
	}
	return "unknown", serverName
}

// NewKialiGatewayToUpstream builds kiali extension attrs for the gateway→upstream edge
func NewKialiGatewayToUpstream(destServerName, statusCode string) KialiExtAttrs {
	destNS, destName := ServerNameParts(destServerName)
	cluster := defaultCluster()
	return KialiExtAttrs{
		Extension:     "mcp-gateway",
		SourceCluster: cluster,
		SourceNS:      defaultSourceNamespace(),
		SourceName:    defaultSourceName(),
		SourceIsRoot:  "true",
		Reporter:      "combined",
		ReporterID:    defaultReporterID(),
		DestCluster:   cluster,
		DestNS:        destNS,
		DestName:      destName,
		Protocol:      "http",
		StatusCode:    statusCode,
		Flags:         "-",
		Secure:        "false",
	}
}

// NewKialiBrokerInbound builds kiali extension attrs for the client→broker edge
func NewKialiBrokerInbound(method string) KialiExtAttrs {
	cluster := defaultCluster()
	return KialiExtAttrs{
		Extension:     "mcp-gateway",
		SourceCluster: cluster,
		SourceNS:      "unknown",
		SourceName:    "mcp-client",
		SourceIsRoot:  "false",
		Reporter:      "combined",
		ReporterID:    defaultReporterID(),
		DestCluster:   cluster,
		DestNS:        "gateway-system",
		DestName:      "mcp-gateway",
		Protocol:      "http",
		StatusCode:    "200",
		Flags:         "-",
		Secure:        "false",
	}
}

func kialiAttrSet(k KialiExtAttrs) attribute.Set {
	return attribute.NewSet(
		attribute.String("extension", k.Extension),
		attribute.String("source_cluster", k.SourceCluster),
		attribute.String("source_namespace", k.SourceNS),
		attribute.String("source_name", k.SourceName),
		attribute.String("source_is_root", k.SourceIsRoot),
		attribute.String("reporter", k.Reporter),
		attribute.String("reporter_id", k.ReporterID),
		attribute.String("dest_cluster", k.DestCluster),
		attribute.String("dest_namespace", k.DestNS),
		attribute.String("dest_name", k.DestName),
		attribute.String("protocol", k.Protocol),
		attribute.String("status_code", k.StatusCode),
		attribute.String("flags", k.Flags),
		attribute.String("secure", k.Secure),
	)
}

// Init initializes all metric instruments from the global MeterProvider.
// Safe to call before a MeterProvider is registered; instruments will use a noop meter
// and begin recording once a real provider is set.
func Init() {
	meter := otel.Meter(meterName)

	ToolCallsTotal, _ = meter.Int64Counter("mcp.tool_calls_total",
		metric.WithDescription("Total number of MCP tool call requests"),
	)

	ToolRouteDuration, _ = meter.Float64Histogram("mcp.tool_route_duration_seconds",
		metric.WithDescription("Duration of router decision for MCP tool calls in seconds"),
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

	KialiExtRequestsTotal, _ = meter.Int64Counter("kiali_ext_requests_total",
		metric.WithDescription("Kiali extension: total requests per source/destination edge"),
	)

	KialiExtResponseTime, _ = meter.Float64Histogram("kiali_ext_response_time_seconds",
		metric.WithDescription("Kiali extension: response time per source/destination edge"),
		metric.WithExplicitBucketBoundaries(kialiResponseTimeBuckets...),
	)
}
