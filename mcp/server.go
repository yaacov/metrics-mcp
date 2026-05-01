// Package mcp registers MCP tools for querying Prometheus/Thanos metrics.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Flags   map[string]any `json:"flags,omitempty" jsonschema:"Command-specific parameters (e.g. query: 'up', output: 'json', namespace: 'mtv-test')"`
}

// MetricsHelpInput is the input schema for the metrics_help tool.
type MetricsHelpInput struct {
	Command string `json:"command,omitempty" jsonschema:"Subcommand or topic to get help for (e.g. query, query_range, discover, labels, preset, promql). Omit for overview."`
}

// CreateServer creates an MCP server with metrics tools registered.
// In HTTP mode the SDK populates req.Extra.Header on every POST with
// that request's HTTP headers, giving each tool call fresh auth credentials.
// In stdio mode there are no HTTP headers and we fall back to CLI defaults.
func CreateServer() *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "kubectl-metrics",
		Version: version.Version,
	}, &mcpsdk.ServerOptions{
		Instructions: "Prometheus / Thanos metrics query server. " +
			"Use metrics_help to discover available preset queries. " +
			"Use metrics_read with command \"discover\" to find metric names. " +
			"Use metrics_read with command \"query\" or \"query_range\" for ad-hoc PromQL queries.",
	})

	registerTools(server)
	return server
}

func registerTools(server *mcpsdk.Server) {
	// ---- metrics_read ----
	// NOTE: The tool description below intentionally omits CLI-only display
	// flags (local_time, no_headers) to keep the LLM context lean. The handler
	// still accepts them — they are just not advertised here.
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "metrics_read",
		Description: `Query Prometheus / Thanos metrics. Use metrics_help for flag details and PromQL reference.

Subcommands (pass as "command"):
  query        Instant PromQL query (flags: query, output, filename, name, group_by, no_pivot, selector)
  query_range  Range PromQL query over a time window (flags: query, name, start, end, step, output, filename, group_by, no_pivot, selector)
               Supports multiple queries: pass query and name as arrays (e.g. query: ["rate(container_cpu_usage_seconds_total[5m])", "container_memory_working_set_bytes"], name: ["cpu", "mem"]).
               Each query's results are labeled with the corresponding name (auto-generated q1, q2, ... if omitted).
               Pass filename to write output to a temp file and return only a summary with the full path (e.g. filename: "data.tsv"). Useful for large results intended for gnuplot or other tools.
  discover     List available metric names (flags: keyword, group_by_prefix)
  labels       List labels or label sets for a metric (flags: metric)
  preset       Run a pre-configured named query (flags: name, namespace, start, end, step, output, filename, group_by, no_pivot, selector)
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
  {command: "query_range", flags: {query: "rate(http_requests_total[5m])", start: "-1h", output: "tsv", filename: "data.tsv"}}
  {command: "query", flags: {query: "up", selector: "namespace=prod,job=~prom.*"}}`,
	}, handleMetricsRead)

	// ---- metrics_help ----
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "metrics_help",
		Description: `Get detailed help for metrics_read subcommands and PromQL query language.

WHEN TO USE: Before calling metrics_read, call metrics_help("<command>") to learn the
available flags and their meaning. Call metrics_help("promql") for PromQL syntax reference.

Commands: query, query_range, discover, labels, preset, promql
Omit command for an overview of all subcommands and available presets.`,
	}, handleMetricsHelp)
}

func handleMetricsRead(ctx context.Context, req *mcpsdk.CallToolRequest, input MetricsReadInput) (*mcpsdk.CallToolResult, any, error) {
	// Validate command early (doesn't require a connection)
	command := strings.TrimSpace(strings.ToLower(input.Command))
	if command == "" {
		return textResult("Missing required field 'command'. Use one of: query, query_range, discover, labels, preset.\nCall metrics_help for details."), nil, nil
	}
	validCommands := map[string]bool{"query": true, "query_range": true, "discover": true, "labels": true, "preset": true}
	if !validCommands[command] {
		return textResult(fmt.Sprintf("Unknown command %q. Available: query, query_range, discover, labels, preset.\nCall metrics_help(\"%s\") for details.", command, command)), nil, nil
	}

	// Extract credentials from request headers into context
	if req.Extra != nil && req.Extra.Header != nil {
		var err error
		ctx, err = connection.WithCredsFromHeaders(ctx, req.Extra.Header)
		if err != nil {
			return textResult(fmt.Sprintf("TLS configuration error: %v", err)), nil, nil
		}
	}

	promURL, rt, err := connection.ResolveConnection(ctx)
	if err != nil {
		return textResult(fmt.Sprintf("TLS configuration error: %v", err)), nil, nil
	}
	if promURL == "" {
		return textResult("Prometheus URL not configured. Provide it via --url flag, X-Metrics-Server header, or ensure cluster access for auto-discovery."), nil, nil
	}
	flags := input.Flags
	if flags == nil {
		flags = map[string]any{}
	}

	// Resolve effective output format: explicit flag, then infer from filename extension.
	outputFormat := metrics.FlagStr(flags, "output")
	fileName := metrics.FlagStr(flags, "filename")
	var fullPath string
	var reservedFile *os.File

	if fileName != "" {
		if fileName == "." || fileName == ".." || filepath.Base(fileName) != fileName || strings.ContainsAny(fileName, `/\`) {
			return textResult(fmt.Sprintf("filename %q must be a plain filename without path separators (e.g. \"data.tsv\")", fileName)), nil, nil
		}
		if outputFormat == "" {
			switch strings.ToLower(filepath.Ext(fileName)) {
			case ".csv":
				outputFormat = "csv"
			case ".tsv":
				outputFormat = "tsv"
			case ".json":
				outputFormat = "json"
			}
		}
		fullPath = filepath.Join(os.TempDir(), fileName)
		f, openErr := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if openErr != nil {
			if os.IsExist(openErr) {
				return textResult(fmt.Sprintf("File %s already exists; use a different filename", fullPath)), nil, nil
			}
			return textResult(fmt.Sprintf("Failed to create file %s: %v", fullPath, openErr)), nil, nil
		}
		reservedFile = f
	}

	client := prometheus.NewClient(promURL, rt)
	t0 := time.Now()
	var result string

	tableOpts := ptable.Options{
		MetricName: metrics.FlagStr(flags, "name"),
		LocalTime:  metrics.FlagBool(flags, "local_time"),
		GroupBy:    metrics.FlagStr(flags, "group_by"),
		NoPivot:    metrics.FlagBool(flags, "no_pivot"),
		NoHeaders:  metrics.FlagBool(flags, "no_headers"),
		Selector:   metrics.FlagStr(flags, "selector"),
	}

	switch command {
	case "query":
		result, err = metrics.Query(ctx, client,
			metrics.FlagStr(flags, "query"),
			outputFormat,
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
			outputFormat,
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
			outputFormat,
			tableOpts)
	default:
		if reservedFile != nil {
			reservedFile.Close()
			os.Remove(fullPath)
		}
		return textResult(fmt.Sprintf("Unknown command %q. Available: query, query_range, discover, labels, preset.\nCall metrics_help(\"%s\") for details.", command, command)), nil, nil
	}

	if err != nil {
		if reservedFile != nil {
			reservedFile.Close()
			os.Remove(fullPath)
		}
		return textResult(metrics.FriendlyError(command, err, promURL)), nil, nil
	}
	klog.V(1).Infof("metrics_read %s completed in %.3fs", command, time.Since(t0).Seconds())

	if reservedFile != nil {
		_, writeErr := reservedFile.WriteString(result)
		closeErr := reservedFile.Close()
		if writeErr != nil || closeErr != nil {
			os.Remove(fullPath)
			if writeErr != nil {
				return textResult(fmt.Sprintf("Failed to write file %s: %v", fullPath, writeErr)), nil, nil
			}
			return textResult(fmt.Sprintf("Failed to close file %s: %v", fullPath, closeErr)), nil, nil
		}

		lines := strings.Split(result, "\n")
		var rows int
		var header string

		switch outputFormat {
		case "json":
			var arr []json.RawMessage
			if json.Unmarshal([]byte(result), &arr) == nil {
				rows = len(arr)
			} else {
				for _, l := range lines {
					if strings.TrimSpace(l) != "" {
						rows++
					}
				}
			}
		case "raw":
			for _, l := range lines {
				if strings.TrimSpace(l) != "" {
					rows++
				}
			}
		case "markdown", "":
			if len(lines) > 0 {
				header = lines[0]
			}
			rows = len(lines) - 1
			if rows > 0 && lines[len(lines)-1] == "" {
				rows--
			}
			for _, l := range lines[1:] {
				bare := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(l), "|", ""), " ", "")
				if len(bare) > 0 && strings.Trim(bare, "-:") == "" {
					rows--
				}
			}
		default:
			if len(lines) > 0 {
				header = lines[0]
			}
			rows = len(lines) - 1
			if rows > 0 && lines[len(lines)-1] == "" {
				rows--
			}
		}

		if header != "" {
			return textResult(fmt.Sprintf("Wrote %d rows to %s\nColumns: %s", rows, fullPath, header)), nil, nil
		}
		return textResult(fmt.Sprintf("Wrote %d items to %s", rows, fullPath)), nil, nil
	}

	return textResult(result), nil, nil
}

func handleMetricsHelp(ctx context.Context, req *mcpsdk.CallToolRequest, input MetricsHelpInput) (*mcpsdk.CallToolResult, any, error) {
	command := strings.TrimSpace(strings.ToLower(input.Command))
	return textResult(help.GenerateHelp(command)), nil, nil
}

// textResult creates a simple text CallToolResult.
func textResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}
}
