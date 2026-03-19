# Metrics & Kiali Investigation

Summary of what was implemented across this branch to add metrics and Kiali support to the MCP Gateway.

## What was done

### Full OTEL observability (traces, logs, metrics)

The OTEL stack previously only handled traces and logs. This branch completed the picture by adding a metrics pipeline and connecting all three signals to visualization tools.

**Metrics pipeline**: Prometheus was deployed in the `observability` namespace with remote write receiver enabled. The OTEL Collector got a new `prometheusremotewrite` exporter and `metrics` pipeline so application metrics flow from the broker-router through OTLP to the Collector and into Prometheus. The OpenTelemetry SDK in `internal/otel/` was extended with a `MetricsProvider` that creates an OTLP metric exporter, wired into `SetupOTelSDK` alongside the existing trace and log providers. All three signals are now enabled when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.

**Istio gateway metrics**: The gateway proxy's Envoy now emits standard Istio Prometheus metrics (`istio_requests_total`, `istio_request_duration_milliseconds`) on port 15020. A Telemetry resource configures the `prometheus` metrics provider, and a Prometheus scrape job discovers the gateway endpoints in `gateway-system`. The `ISTIO_METRICS=1` Makefile flag patches the Istio CR to set `defaultProviders.metrics: ["prometheus"]`.

**Custom MCP metrics**: Six application-level instruments defined in `internal/metrics/metrics.go` using OpenTelemetry (not `prometheus/client_golang`):

| Metric | Type | Where recorded |
|--------|------|----------------|
| `mcp.tool_calls_total` | Counter | Router `HandleToolCall` |
| `mcp.tool_route_duration_seconds` | Histogram | Router `HandleToolCall` |
| `mcp.requests_total` | Counter | Router + Broker (by component) |
| `mcp.active_sessions` | UpDownCounter | Broker session hooks |
| `mcp.server_health` | UpDownCounter | Upstream manager |
| `mcp.tool_list_total` | Counter | Broker `AfterListTools` hook |

Instruments use the global `otel.Meter("mcp-gateway")` and record via noop until a real MeterProvider is registered at startup.

**Grafana dashboard**: A pre-provisioned "MCP Gateway Metrics" dashboard (`examples/otel/grafana-dashboards.yaml`) with panels for tool calls, tool list requests, route duration, active sessions, tool calls per server/tool, healthy servers, call rate over time, and requests by method.

### Kiali integration

**Kiali operator** (`build/kiali.mk`): Helm-based install of kiali-operator v2.3 configured with:
- Anonymous auth for dev environments
- Prometheus at `http://prometheus.observability:9090` (in-cluster for server-side queries)
- Grafana with split URLs: `http://localhost:3000` for browser redirects, `http://grafana.observability:3000` for server-side queries
- Tempo tracing with split URLs: `http://localhost:3000/explore` for browser redirects (opens Grafana Explore), `http://tempo.observability:3200` for server-side trace queries
- Gateway API class discovery (`class_name: istio`)

Kiali has two URL fields for external services: `url` for browser redirects (must be reachable from the user's browser) and `in_cluster_url` for server-side API calls (must be reachable from inside the cluster). Setting both to the in-cluster URL causes broken browser links.

### Kiali configuration reference

All settings are in `build/kiali.mk` and applied via Helm `--set` values.

**General:**

| Setting | Value | Purpose |
|---------|-------|---------|
| `cr.create` | `true` | Helm creates the Kiali CR automatically |
| `cr.namespace` | `istio-system` | Deploy Kiali in the Istio namespace |
| `cr.spec.auth.strategy` | `anonymous` | No login required (dev environments only) |
| `cr.spec.deployment.cluster_wide_access` | `true` | Kiali can see all namespaces |

**Prometheus:**

| Setting | Value | Purpose |
|---------|-------|---------|
| `external_services.prometheus.url` | `http://prometheus.observability:9090` | In-cluster URL for Kiali server-side Prometheus queries (Istio metrics, health calculations, graph construction) |

**Grafana:**

| Setting | Value | Purpose |
|---------|-------|---------|
| `external_services.grafana.enabled` | `true` | Enable Grafana integration |
| `external_services.grafana.url` | `http://localhost:3000` | Browser-facing URL for Grafana links (must be port-forwarded) |
| `external_services.grafana.in_cluster_url` | `http://grafana.observability:3000` | In-cluster URL for Kiali server-side Grafana API calls |

**Tracing (Tempo):**

| Setting | Value | Purpose |
|---------|-------|---------|
| `external_services.tracing.enabled` | `true` | Enable distributed tracing integration |
| `external_services.tracing.provider` | `tempo` | Use Grafana Tempo as the tracing backend |
| `external_services.tracing.url` | `http://localhost:3000/explore` | Browser-facing URL for trace links (opens Grafana Explore, must be port-forwarded) |
| `external_services.tracing.in_cluster_url` | `http://tempo.observability:3200` | In-cluster URL for Kiali server-side Tempo API queries |
| `external_services.tracing.use_grpc` | `false` | Use HTTP API instead of gRPC for Tempo queries |
| `external_services.tracing.tempo_config.org_id` | `1` | Tempo tenant ID (single-tenant setup) |
| `external_services.tracing.tempo_config.datasource_uid` | `tempo` | Grafana Tempo datasource UID, used to construct correct Explore links |
| `external_services.tracing.tempo_config.url_format` | `grafana` | Format trace links as Grafana Explore URLs |
| `external_services.tracing.namespace_selector` | `true` | Scope trace queries to the selected namespace to reduce noise |

**Istio:**

| Setting | Value | Purpose |
|---------|-------|---------|
| `external_services.istio.gateway_api_classes[0].class_name` | `istio` | Discover Gateway API resources using the `istio` GatewayClass |

**OTEL service name** (in `Makefile`, not kiali.mk):

| Setting | Value | Purpose |
|---------|-------|---------|
| `OTEL_SERVICE_NAME` | `mcp-broker-router` | Set on the broker deployment via `kubectl set env`. Must match the Kubernetes deployment name so Kiali can correlate traces from Tempo with the workload in the Traces tab. Default was `mcp-gateway` which caused a mismatch. |

### Makefile targets

The `make otel` target deploys the full stack and accepts feature flags:

```
make otel ISTIO_METRICS=1 ISTIO_TRACING=1 KIALI=1
```

Supporting targets: `make otel-forward` (Grafana 3000, Prometheus 9090), `make otel-delete`, `make otel-status`, `make kiali-install`, `make kiali-uninstall`, `make kiali-forward` (20001).

The `otel` target restarts Prometheus after applying manifests so ConfigMap changes take effect immediately.

## Kiali in gateway-only mode

MCP Gateway uses Istio only as a Gateway API provider -- no sidecars, no service mesh. This limits what Kiali can show since it was designed for full mesh observability.

### What works

**Traffic Graph** (most useful view):
- Select `gateway-system` + `mcp-test` namespaces, set graph type to "App graph"
- Istio metrics show client-to-gateway edges from the gateway proxy
- Kiali extension metrics add gateway-to-upstream-MCP-server edges, showing per-server
  request rates, error rates, and response times
- Combined view shows the full request path from clients through the gateway to individual
  upstream MCP servers
- Note: use "App graph" not "Workload graph" (workload graph rejects extension service nodes)

**Istio Config**:
- Lists Gateway and HTTPRoute resources with validation badges
- Flags misconfigurations such as HTTPRoutes referencing non-existent services or gateways
- Validates listener hostname/port uniqueness on Gateways

**Workloads** (gateway-system namespace):
- `mcp-gateway` workload detail page shows inbound/outbound request rate, error rate, and duration from Prometheus
- Metrics tab provides time-series charts for the gateway workload

**Overview page**:
- Set "Health for" to **Outbound** to see gateway traffic
- Default "Inbound" view shows "No inbound traffic" for all namespaces -- this is expected without sidecars

**Distributed Tracing**:
- Links to Grafana Explore page for Tempo trace browsing (requires `make otel-forward` for port 3000)
- Requires `ISTIO_TRACING=1` to be set for gateway proxy traces to appear

### What does not work (inherent limitations)

These are limitations of a gateway-only Istio deployment. No configuration changes can fix them without adding sidecars or ambient mesh:

- **Overview "Inbound" always empty**: no `reporter="destination"` metrics exist without sidecar proxies on backend pods
- **Destination labels show as `"unknown"`**: gateway-side metrics lack destination pod/workload identity
- **No mTLS indicators**: no sidecar-to-sidecar connections means no mTLS status
- **No service-to-service topology**: only one hop visible (gateway to backend), no deeper service mesh graph
- **No workload health for backends**: only the gateway workload has metrics

## Kiali extension framework investigation

The Kiali extension framework (introduced in v2.0, KEP: kiali/kiali#7485) allows third-party traffic
metrics to appear in Kiali's service mesh graph. Extensions emit standardised `kiali_ext_*` metrics
with source/destination labels, and Kiali's extensions graph appender merges them into the traffic
graph alongside standard Istio metrics.

### What was implemented

Two Kiali extension metrics were added to `internal/metrics/metrics.go`:

| Metric | Type | Purpose |
|--------|------|---------|
| `kiali_ext_requests_total` | Counter | Request count per source/destination edge |
| `kiali_ext_response_time_seconds` | Histogram | Latency with Kiali-specified buckets (.005-.01-.025-.05-.1-.25-.5-1-2.5-5-10) |

Each metric carries 14 labels per the Kiali extension spec: `extension`, `source_cluster`,
`source_namespace`, `source_name`, `source_is_root`, `reporter`, `reporter_id`, `dest_cluster`,
`dest_namespace`, `dest_name`, `protocol`, `status_code`, `flags`, `secure`.

The router's `HandleToolCall` records both metrics on every tool call with:
- Source: the gateway workload (`source_is_root=true` so the appender reuses the existing Istio node)
- Destination: the upstream MCP server, with namespace/name parsed from the `MCPServer.Name` field

The extension was registered in the Kiali CR via Helm in `build/kiali.mk`:
```yaml
spec:
  extensions:
    - enabled: true
      name: mcp-gateway
```

### Issues encountered

**Issue 1: Cluster name mismatch prevents root node matching**

The extension metrics initially used `source_cluster=local` as the default cluster identifier.
Kiali/Istio defaults to `Kubernetes` as the cluster name. The extensions appender's `findRootNode`
function matches on cluster + namespace + name to find the existing gateway workload node. With
mismatched cluster names, the lookup failed and the appender created a new service-type node
instead of reusing the existing workload node.

Fix: changed default cluster to `Kubernetes` (overridable via `KIALI_CLUSTER` env var).

**Issue 2: Source name mismatch prevents root node matching**

The extension metrics used `source_name=mcp-gateway` but the Istio gateway deployment is named
`mcp-gateway-istio` (Istio's naming convention: `<gateway-name>-istio`). The `findRootNode`
function checks `n.Workload == name || n.Service == name || n.App == name`, so the name must
match one of these fields on the existing Istio workload node.

Fix: changed default source name to `mcp-gateway-istio` (overridable via `KIALI_EXT_SOURCE_NAME`).

**Issue 3: Extensions appender creates service-type nodes that break workload graphs**

The extensions appender in `graph/telemetry/istio/appender/extensions.go` calls:
```go
graph.Id(cluster, namespace, name, "", "", "", "", graphType)
```

This passes `name` into the `service` parameter with empty workload/app/version. The `Id()`
function returns `NodeTypeService` for workload graphs when workload is empty. The workload
graph renderer then fails with:

```
Cannot load the graph: Expected nodeType [workload] for node
[&{ID:svc_Kubernetes_mcp-test_test-server2 NodeType:service ...}]
```

This affects **destination nodes** (upstream MCP servers). The source node is handled correctly
via `findRootNode` (after fixes 1 and 2), but destination nodes have no existing workload in the
Istio graph to match against. The appender always creates them as service-type nodes.

This is a bug in the Kiali extensions appender -- it should either pass `name` as the workload
parameter or the graph renderer should tolerate service-type nodes from extensions. The Kiali
documentation states "Kiali will always show a terminal service node when the request itself
fails to be routed to a destination workload", but the workload graph validation rejects
extension-created service nodes.

**Issue 4: App graph frontend rendering crash (resolved)**

The app graph crashed with:
```
Cannot convert undefined or null to object
```

The Kiali frontend's `getEdgeHealth` calls `transformEdgeResponses` which calls `Object.values()`
on a `flags` field in the edge traffic response. Istio edges include `flags: {"-": "100.0"}` but
extension edges were missing the `flags` field entirely because the extension metrics had
`Flags: ""` (empty string). The Kiali extensions appender only populates the `flags` map when the
label value is non-empty.

Istio uses `"-"` as the flags label value to mean "no flags". Changing the extension metrics to
use `Flags: "-"` causes the appender to produce the expected `flags: {"-": "100.0"}` structure,
which fixes the frontend crash.

Fix: changed `Flags` from `""` to `"-"` in both `NewKialiGatewayToUpstream` and
`NewKialiBrokerInbound`.

### Current status

The Kiali extension framework is working for the app graph type. The extension metrics produce
edges in the Kiali traffic graph showing gateway-to-upstream-MCP-server traffic with request
rates, error rates, and response times.

**Working (app graph)**:
- Extension edges render correctly alongside Istio edges
- Source node (gateway) is matched to the existing Istio workload node via `findRootNode`
- Destination nodes (upstream MCP servers) appear as service-type nodes
- Edge health indicators (traffic coloring) work correctly

**Not working (workload graph)**:
- Issue 3 remains: the extensions appender creates service-type destination nodes that the
  workload graph renderer rejects. Use the app graph type instead.

## Possible improvements

Areas where Kiali could provide better observability for gateway-only Istio deployments.

### Trace-based topology

Kiali builds its traffic graph exclusively from Prometheus Istio metrics (`istio_requests_total`).
In a gateway-only setup this limits the graph to one hop (gateway to backend) with destination
labels showing as `"unknown"`.

The MCP Gateway already produces distributed traces via its OTEL SDK that capture the full
request flow: client to gateway to router to broker to upstream MCP server. These traces are
stored in Tempo and contain service names, operation names, and parent-child span relationships
that describe the complete topology.

If Kiali could build topology from tracing data (Tempo/Jaeger) in addition to Prometheus metrics,
it would show the full request path through the gateway components and into backend servers
without requiring sidecars or mesh telemetry. The span attributes already contain the service
identities and request metadata needed to construct graph edges.

### Gateway-only deployment mode

Kiali assumes either a full sidecar mesh. In a gateway-only deployment it shows
"missing sidecar" warnings on all workloads and requires `reporter="destination"` metrics that
will never exist.

A gateway-only mode would:
- Suppress "missing sidecar" warnings for workloads not expected to have sidecars
- Fall back to `reporter="source"` metrics as authoritative for all views
- Derive destination identity from available request headers or trace context instead of
  requiring sidecar-reported labels

Istio itself has an open request for gateway-only mode support (istio/istio#58456). Kiali
awareness of this deployment pattern would complement that effort.

### Configurable metric labels for destination identity

Kiali's traffic graph is hardcoded to read destination identity from standard Istio labels on
`istio_requests_total` (`destination_service_name`, `destination_workload`, etc.). These labels
are populated by sidecar proxies on the destination pod. Without sidecars they show as `"unknown"`.

In the MCP Gateway, the ext_proc router knows the destination server name, tool name, and
routing path for every request. The Istio Telemetry API can add custom labels to
`istio_requests_total` via `tagOverrides` (e.g., `request_host` from the `:authority` header).
This puts destination identity into Prometheus, but Kiali ignores it because it only reads the
hardcoded label names.

If Kiali allowed configuring which metric labels to use for graph edge construction, any
gateway-only deployment could map its own labels into the traffic graph. For example, mapping
`request_host` to destination service identity would give each MCP server hostname its own
edge with independent rate and error metrics.

### HTTPRoute-driven topology

Kiali already reads Gateway and HTTPRoute resources for configuration validation. These
resources explicitly define the routing topology: which hostnames route to which backend
services, with what path matching and filters (URLRewrite, RequestHeaderModifier).

Currently this information is only used for validation badges in the Istio Config view. If
Kiali used HTTPRoute rules to enrich graph edges, it could show which route matched a request,
what filters were applied, and which backend received the traffic. This would provide topology
context that is independent of metrics entirely -- the routing structure is declared in the
Gateway API resources themselves.

For the MCP Gateway this would mean the traffic graph could show edges from the gateway to
each backend service based on the HTTPRoute definitions, annotated with any URLRewrite filters
(e.g., rewriting to external hostnames like `api.githubcopilot.com`). Combined with the
source-side request rate metrics from the gateway proxy, this would produce a meaningful graph
without requiring any sidecars or additional metric labels.