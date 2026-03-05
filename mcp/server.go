// Package mcp registers MCP tools for querying Prometheus/Thanos metrics.
package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yaacov/kubectl-metrics/pkg/connection"
	"github.com/yaacov/kubectl-metrics/pkg/help"
	"github.com/yaacov/kubectl-metrics/pkg/metrics"
	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
	"github.com/yaacov/kubectl-metrics/pkg/version"
	"k8s.io/klog/v2"
)

// MetricsReadInput is the input schema for the metrics_read tool.
type MetricsReadInput struct {
	Command string         `json:"command" jsonschema:"Subcommand: query | query_range | discover | labels | preset"`
	Flags   map[string]any `json:"flags,omitempty" jsonschema:"Command-specific parameters (e.g. query: 'up', format: 'json', namespace: 'mtv-test')"`
}

// MetricsHelpInput is the input schema for the metrics_help tool.
type MetricsHelpInput struct {
	Command string `json:"command,omitempty" jsonschema:"Subcommand or topic to get help for (e.g. query, query_range, discover, labels, preset, promql). Omit for overview."`
}

// CreateServer creates an MCP server with metrics tools registered.
// capturedHeaders are injected into every tool call's context for
// transparent per-session authentication.
func CreateServer(capturedHeaders http.Header) *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "kubectl-metrics",
		Version: version.Version,
	}, &mcpsdk.ServerOptions{
		Instructions: "Prometheus / Thanos metrics query server. " +
			"Use metrics_help to discover available preset queries. " +
			"Use metrics_read with command \"discover\" to find metric names. " +
			"Use metrics_read with command \"query\" or \"query_range\" for ad-hoc PromQL queries.",
	})

	registerTools(server, capturedHeaders)
	return server
}

func registerTools(server *mcpsdk.Server, capturedHeaders http.Header) {
	// ---- metrics_read ----
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "metrics_read",
		Description: `Query Prometheus / Thanos metrics. Use metrics_help for flag details and PromQL reference.

Subcommands (pass as "command"):
  query        Instant PromQL query (flags: query, format, name, local_time, group_by, no_pivot, selector)
  query_range  Range PromQL query over a time window (flags: query, name, start, end, step, format, local_time, group_by, no_pivot, selector)
               Supports multiple queries: pass query and name as arrays (e.g. query: ["rate(container_cpu_usage_seconds_total[5m])", "container_memory_working_set_bytes"], name: ["cpu", "mem"]).
               Each query's results are labeled with the corresponding name (auto-generated q1, q2, ... if omitted).
  discover     List available metric names (flags: keyword, group_by_prefix)
  labels       List labels or label sets for a metric (flags: metric)
  preset       Run a pre-configured named query (flags: name, namespace, start, end, step, format, local_time, group_by, no_pivot, selector)
               Every preset works as both instant (default) and range query. Pass start to get a time-series trend.
               Range queries use a pivot table by default (one column per label combination). Set no_pivot: true
               to revert to the traditional row-per-sample format.
               Use selector to filter results by labels post-query (e.g. "namespace=prod,pod=~nginx.*").
               Supported operators: = (equal), != (not equal), =~ (regex), !~ (negative regex).

Examples:
  {command: "query", flags: {query: "up"}}
  {command: "query_range", flags: {query: "rate(http_requests_total[5m])", start: "-1h"}}
  {command: "query_range", flags: {query: ["sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)", "sum(container_memory_working_set_bytes) by (namespace)"], name: ["cpu", "mem"], start: "-1h"}}
  {command: "discover", flags: {keyword: "mtv", group_by_prefix: true}}
  {command: "labels", flags: {metric: "container_network_receive_bytes_total"}}
  {command: "preset", flags: {name: "cluster_cpu_utilization"}}
  {command: "preset", flags: {name: "mtv_migration_status", namespace: "mtv-test", group_by: "namespace"}}
  {command: "preset", flags: {name: "mtv_net_throughput"}}
  {command: "preset", flags: {name: "mtv_net_throughput", start: "-2h", step: "30s"}}
  {command: "query", flags: {query: "up", selector: "namespace=prod,job=~prom.*"}}`,
	}, wrapWithHeaders(handleMetricsRead, capturedHeaders))

	// ---- metrics_help ----
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "metrics_help",
		Description: `Get detailed help for metrics_read subcommands and PromQL query language.

WHEN TO USE: Before calling metrics_read, call metrics_help("<command>") to learn the
available flags and their meaning. Call metrics_help("promql") for PromQL syntax reference.

Commands: query, query_range, discover, labels, preset, promql
Omit command for an overview of all subcommands and available presets.`,
	}, wrapWithHeaders(handleMetricsHelp, capturedHeaders))
}

func handleMetricsRead(ctx context.Context, req *mcpsdk.CallToolRequest, input MetricsReadInput) (*mcpsdk.CallToolResult, struct{}, error) {
	// Extract credentials from request headers into context
	if req.Extra != nil && req.Extra.Header != nil {
		ctx = connection.WithCredsFromHeaders(ctx, req.Extra.Header)
	}

	promURL, rt := connection.ResolveConnection(ctx)
	if promURL == "" {
		return textResult("Prometheus URL not configured. Provide it via --url flag, X-Metrics-Server header, or ensure cluster access for auto-discovery."), struct{}{}, nil
	}

	command := strings.TrimSpace(strings.ToLower(input.Command))
	if command == "" {
		return textResult("Missing required field 'command'. Use one of: query, query_range, discover, labels, preset.\nCall metrics_help for details."), struct{}{}, nil
	}
	flags := input.Flags
	if flags == nil {
		flags = map[string]any{}
	}

	client := prometheus.NewClient(promURL, rt)
	t0 := time.Now()
	var result string
	var err error

	tableOpts := ptable.Options{
		MetricName: metrics.FlagStr(flags, "name"),
		LocalTime:  metrics.FlagBool(flags, "local_time"),
		GroupBy:    metrics.FlagStr(flags, "group_by"),
		NoPivot:    metrics.FlagBool(flags, "no_pivot"),
		Selector:   metrics.FlagStr(flags, "selector"),
	}

	switch command {
	case "query":
		result, err = metrics.Query(ctx, client,
			metrics.FlagStr(flags, "query"),
			metrics.FlagStr(flags, "output"),
			tableOpts)
	case "query_range":
		queries := metrics.BuildNamedQueries(
			metrics.FlagStrSlice(flags, "query"),
			metrics.FlagStrSlice(flags, "name"),
		)
		result, err = metrics.QueryRangeMulti(ctx, client,
			queries,
			metrics.FlagStr(flags, "start"),
			metrics.FlagStr(flags, "end"),
			metrics.FlagStr(flags, "step"),
			metrics.FlagStr(flags, "output"),
			tableOpts)
	case "discover":
		result, err = metrics.Discover(ctx, client,
			metrics.FlagStr(flags, "keyword"),
			metrics.FlagBool(flags, "group_by_prefix"))
	case "labels":
		result, err = metrics.Labels(ctx, client, metrics.FlagStr(flags, "metric"))
	case "preset":
		result, err = metrics.Preset(ctx, client,
			metrics.FlagStr(flags, "name"),
			metrics.FlagStr(flags, "namespace"),
			metrics.FlagStr(flags, "start"),
			metrics.FlagStr(flags, "end"),
			metrics.FlagStr(flags, "step"),
			metrics.FlagStr(flags, "output"),
			tableOpts)
	default:
		return textResult(fmt.Sprintf("Unknown command %q. Available: query, query_range, discover, labels, preset.\nCall metrics_help(\"%s\") for details.", command, command)), struct{}{}, nil
	}

	if err != nil {
		return textResult(metrics.FriendlyError(command, err, promURL)), struct{}{}, nil
	}
	klog.V(1).Infof("metrics_read %s completed in %.3fs", command, time.Since(t0).Seconds())
	return textResult(result), struct{}{}, nil
}

func handleMetricsHelp(ctx context.Context, req *mcpsdk.CallToolRequest, input MetricsHelpInput) (*mcpsdk.CallToolResult, struct{}, error) {
	command := strings.TrimSpace(strings.ToLower(input.Command))
	return textResult(help.GenerateHelp(command)), struct{}{}, nil
}

// textResult creates a simple text CallToolResult.
func textResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}
}

// wrapWithHeaders wraps a tool handler to inject captured HTTP headers into RequestExtra.
func wrapWithHeaders[In, Out any](
	handler func(context.Context, *mcpsdk.CallToolRequest, In) (*mcpsdk.CallToolResult, Out, error),
	headers http.Header,
) func(context.Context, *mcpsdk.CallToolRequest, In) (*mcpsdk.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest, input In) (*mcpsdk.CallToolResult, Out, error) {
		if req.Extra == nil && headers != nil {
			req.Extra = &mcpsdk.RequestExtra{Header: headers}
		} else if req.Extra != nil && req.Extra.Header == nil && headers != nil {
			req.Extra.Header = headers
		}
		return handler(ctx, req, input)
	}
}
