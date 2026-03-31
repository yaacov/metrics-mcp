// Package help provides help text for metrics commands and PromQL reference.
package help

import (
	"fmt"
	"strings"

	"github.com/yaacov/kubectl-metrics/pkg/presets"
)

// GenerateHelp returns help text for the given command or topic.
// Pass an empty string for an overview of all commands.
//
// NOTE: This help is consumed by LLMs via the MCP metrics_help tool.
// CLI-only flags (local_time, no_headers) are intentionally omitted to keep
// the LLM context lean. These flags still work if passed — they are just
// not advertised here. See the CLI --help for the full flag set.
func GenerateHelp(command string) string {
	switch command {
	case "query":
		return `metrics_read "query" — Execute an instant PromQL query

Returns the current value of the expression at a single point in time.

Flags:
  query    (required)  PromQL expression (e.g. "up", "node_cpu_seconds_total")
  output   (optional)  Output format: markdown (default), table, json, raw, csv, tsv
  file     (optional)  MCP only. Write output to a file and return only a summary. Path must be under the system temp directory (e.g. "/tmp/data.tsv")
  name     (optional)  Metric name for the first table column (useful when __name__ is absent)
  group_by (optional)  Label name to split results into sub-tables (e.g. "namespace", "pod")
  no_pivot (optional)  Disable pivot table layout for range results (default: false)
  selector (optional)  Label selector to filter results post-query (e.g. "namespace=prod,pod=~nginx.*")
                        Operators: = (equal), != (not equal), =~ (regex match), !~ (negative regex)

Examples:
  {command: "query", flags: {query: "up"}}
  {command: "query", flags: {query: "up", selector: "namespace=prod"}}
  {command: "query", flags: {query: "sum(rate(http_requests_total[5m])) by (code)", output: "markdown"}}
  {command: "query", flags: {query: "node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes * 100"}}`

	case "query_range":
		return `metrics_read "query_range" — Execute one or more range PromQL queries

Returns values over a time window (default: last 1 hour, 60s steps).
Supports multiple queries in a single call: pass query and name as arrays.
Each query's results have __name__ set to the corresponding name (or auto-generated q1, q2, ...).
Range queries use a pivot table by default (one column per name/label combination,
one row per timestamp). Set no_pivot: true to revert to the traditional
row-per-sample format.

Flags:
  query    (required)  PromQL expression, or array of expressions for multi-query
  name     (optional)  Display name for each query (string or array). Auto-generated as q1, q2, ... if omitted.
  start    (optional)  Start time: ISO-8601, Unix epoch, or relative offset (default: -1h)
                        Relative offsets: -30m, -1h, -7d, -2w
  end      (optional)  End time: same formats as start (default: now)
  step     (optional)  Step interval (default: 60s). Use "15s", "5m", "1h", etc.
  output   (optional)  Output format: markdown (default), table, json, raw, csv, tsv
  file     (optional)  MCP only. Write output to a file and return only a summary. Path must be under the system temp directory (e.g. "/tmp/data.tsv").
                        Useful for large results intended for gnuplot or other tools.
  group_by (optional)  Label name to split results into sub-tables (e.g. "namespace", "pod")
  no_pivot (optional)  Disable pivot table layout (default: false)
  selector (optional)  Label selector to filter results post-query (e.g. "namespace=prod,pod=~nginx.*")
                        Operators: = (equal), != (not equal), =~ (regex match), !~ (negative regex)

Examples:
  {command: "query_range", flags: {query: "rate(http_requests_total[5m])", start: "-1h"}}
  {command: "query_range", flags: {query: ["sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)", "sum(container_memory_working_set_bytes) by (namespace)"], name: ["cpu", "mem"], start: "-1h"}}
  {command: "query_range", flags: {query: "rate(http_requests_total[5m])", start: "-1h", output: "tsv", file: "/tmp/data.tsv"}}
  {command: "query_range", flags: {query: "node_cpu_seconds_total", start: "-7d", step: "1h", output: "markdown"}}
  {command: "query_range", flags: {query: "rate(http_requests_total[5m])", start: "-1h", no_pivot: true}}`

	case "discover":
		return `metrics_read "discover" — List available Prometheus metric names

Retrieves all metric names from Prometheus. Optionally filter by keyword or group by prefix.

Flags:
  keyword          (optional)  Case-insensitive substring filter
  group_by_prefix  (optional)  Boolean. Group metrics by two-part prefix and return counts

Examples:
  {command: "discover"}
  {command: "discover", flags: {keyword: "mtv"}}
  {command: "discover", flags: {keyword: "network", group_by_prefix: true}}`

	case "labels":
		return `metrics_read "labels" — List Prometheus labels

Without a metric name: returns all known label names.
With a metric name: returns the label keys that appear on series for that metric.

Flags:
  metric  (optional)  Metric name to inspect. Omit to list all label names.

Examples:
  {command: "labels"}
  {command: "labels", flags: {metric: "container_network_receive_bytes_total"}}`

	case "preset":
		lines := []string{
			`metrics_read "preset" — Run a pre-configured PromQL query by name`,
			"",
			"Presets provide quick access to common cluster monitoring and MTV/Forklift",
			"migration queries without writing PromQL. Many presets support namespace filtering.",
			"",
			"Every preset works as both an instant query (default) and a range query.",
			"Pass start to get a time-series trend (e.g. start: \"-1h\").",
			"Range queries use a pivot table by default (one column per label combination).",
			"Set no_pivot: true to revert to the traditional row-per-sample format.",
			"",
			"Flags:",
			"  name      (required)  Preset name (see list below)",
			"  namespace (optional)  Namespace filter",
			"  start     (optional)  Start time: ISO-8601, Unix epoch, or relative offset (e.g. -1h). Enables range query.",
			"  end       (optional)  End time: same formats as start (default: now)",
			"  step      (optional)  Step interval (default: 60s). Use 15s, 5m, 1h, etc.",
			"  output    (optional)  Output format: markdown (default), table, json, raw, csv, tsv",
			`  file      (optional)  MCP only. Write output to a file and return only a summary. Path must be under the system temp directory (e.g. "/tmp/data.tsv")`,
			"  group_by  (optional)  Label name to split results into sub-tables (e.g. \"namespace\", \"pod\")",
			"  no_pivot  (optional)  Disable pivot table layout (default: false)",
			`  selector  (optional)  Label selector to filter results post-query (e.g. "namespace=prod,pod=~nginx.*")`,
			`                         Operators: = (equal), != (not equal), =~ (regex match), !~ (negative regex)`,
			"",
			"Available presets:",
		}
		for _, p := range presets.ListPresets() {
			lines = append(lines, fmt.Sprintf("  %-40s %s", p.Name, p.Description))
		}
		lines = append(lines, "", "Examples:",
			`  {command: "preset", flags: {name: "cluster_cpu_utilization"}}`,
			`  {command: "preset", flags: {name: "cluster_pod_status"}}`,
			`  {command: "preset", flags: {name: "mtv_migration_status", namespace: "mtv-test"}}`,
			`  {command: "preset", flags: {name: "mtv_migration_pod_rx", namespace: "mtv-test", output: "markdown"}}`,
			`  {command: "preset", flags: {name: "mtv_net_throughput"}}`,
			`  {command: "preset", flags: {name: "mtv_net_throughput", start: "-2h", step: "30s"}}`,
			`  {command: "preset", flags: {name: "cluster_cpu_utilization", start: "-1h"}}`,
		)
		return strings.Join(lines, "\n")

	case "promql":
		return `PromQL Quick Reference
=====================

PromQL (Prometheus Query Language) is used to select and aggregate time-series data.

SELECTORS
  up                                    Metric name (instant vector)
  up{job="prometheus"}                  Label matcher (exact)
  up{job=~"prom.*"}                     Label matcher (regex)
  up{job!="prometheus"}                 Negative matcher
  up{job!~"test.*"}                     Negative regex

RANGE VECTORS (for rate/increase)
  http_requests_total[5m]               Last 5 minutes of samples
  http_requests_total[1h]               Last 1 hour

FUNCTIONS
  rate(metric[5m])                      Per-second rate of increase (for counters)
  irate(metric[5m])                     Instant rate (last two samples)
  increase(metric[1h])                  Total increase over range
  sum(metric)                           Sum across all series
  avg(metric)                           Average across all series
  min(metric) / max(metric)             Min/Max
  count(metric)                         Count of series
  topk(10, metric)                      Top 10 series by value
  bottomk(5, metric)                    Bottom 5 series by value
  sort_desc(metric)                     Sort descending
  absent(metric)                        Returns 1 if metric has no series

AGGREGATION (by / without)
  sum by (namespace)(metric)            Sum grouped by namespace
  avg by (pod)(rate(cpu[5m]))           Avg rate grouped by pod
  sum without (instance)(metric)        Sum dropping instance label

OPERATORS
  metric_a + metric_b                   Addition
  metric_a / metric_b                   Division
  metric_a > 100                        Filter: keep series > 100
  metric_a and metric_b                 Intersection
  metric_a or metric_b                  Union

COMMON PATTERNS
  rate(counter[5m])                     Per-second rate from a counter
  sum by (ns)(rate(bytes_total[5m]))    Aggregate rate by namespace
  histogram_quantile(0.99, rate(h[5m])) P99 latency from histogram
  changes(metric[1h])                   Number of value changes
  delta(metric[1h])                     Difference over range (gauges)
  predict_linear(metric[1h], 3600)      Linear prediction 1h ahead

TIME UNITS
  s = seconds, m = minutes, h = hours, d = days, w = weeks
  Used in range vectors: [5m], [1h], [7d]
  Used in start/end flags: -30m, -1h, -7d, -2w`

	default:
		// Overview: list all commands + presets
		lines := []string{
			"metrics_read — Query Prometheus / Thanos metrics",
			"",
			"SUBCOMMANDS (pass as \"command\"):",
			"  query        Instant PromQL query                    (flags: query, output, file, name, group_by, no_pivot, selector)",
			"  query_range  Range query over a time window          (flags: query, name, start, end, step, output, file, group_by, no_pivot, selector)",
			"  discover     List available metric names              (flags: keyword, group_by_prefix)",
			"  labels       List labels or label sets for a metric  (flags: metric)",
			"  preset       Run a pre-configured named query        (flags: name, namespace, start, end, step, output, file, group_by, no_pivot, selector)",
			"",
			"Call metrics_help(\"<command>\") for detailed flag descriptions and examples.",
			"Call metrics_help(\"promql\") for PromQL query language reference.",
			"",
			"AVAILABLE PRESETS (for use with \"preset\" command):",
		}
		for _, p := range presets.ListPresets() {
			lines = append(lines, fmt.Sprintf("  %-40s %s", p.Name, p.Description))
		}
		lines = append(lines, "",
			"QUICK EXAMPLES:",
			`  {command: "query", flags: {query: "up"}}`,
			`  {command: "query_range", flags: {query: "rate(http_requests_total[5m])", start: "-1h"}}`,
			`  {command: "discover", flags: {keyword: "mtv"}}`,
			`  {command: "preset", flags: {name: "cluster_cpu_utilization"}}`,
			`  {command: "preset", flags: {name: "mtv_migration_status"}}`,
			`  {command: "preset", flags: {name: "mtv_net_throughput"}}`,
		)
		return strings.Join(lines, "\n")
	}
}
