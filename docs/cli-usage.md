# CLI Usage

kubectl-metrics provides subcommands for querying Prometheus / Thanos metrics from the command line.

## Commands

### discover

List available Prometheus metric names.

```bash
# List all metrics
kubectl-metrics discover

# Filter by keyword
kubectl-metrics discover --keyword mtv

# Group by prefix with counts
kubectl-metrics discover --keyword network --group-by-prefix
```

### query

Execute an instant PromQL query (returns current values).

```bash
kubectl-metrics query --query "up"
kubectl-metrics query --query "sum(rate(http_requests_total[5m])) by (status)" --name http_rps
kubectl-metrics query --query "node_memory_MemAvailable_bytes" --format json
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--query` | (required) | PromQL expression |
| `--format` | `table` | Output format: `table`, `markdown`, `json`, `raw` |
| `--name` | | Metric name for the first table column (useful for aggregate queries that lack `__name__`) |
| `--local-time` | `false` | Display timestamps in local timezone instead of UTC |
| `--group-by` | | Label name to split results into sub-tables (e.g. `namespace`, `pod`) |

### query-range

Execute a range PromQL query over a time window.

```bash
kubectl-metrics query-range --query "rate(http_requests_total[5m])" --start "-1h"
kubectl-metrics query-range --query "node_cpu_seconds_total" --start "-7d" --step "1h" --format json
kubectl-metrics query-range --query "sum(rate(http_requests_total[5m])) by (status)" --start "-1h" --name http_rps
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--query` | (required) | PromQL expression |
| `--start` | `-1h` | Start time: ISO-8601, Unix epoch, or relative (`-1h`, `-7d`) |
| `--end` | `now` | End time (same formats) |
| `--step` | `60s` | Query resolution step |
| `--format` | `table` | Output format: `table`, `markdown`, `json`, `raw` |
| `--name` | | Metric name for the first table column (useful for aggregate queries that lack `__name__`) |
| `--local-time` | `false` | Display timestamps in local timezone instead of UTC |
| `--group-by` | | Label name to split results into sub-tables (e.g. `namespace`, `pod`) |

### labels

List Prometheus label names, optionally scoped to a metric.

```bash
# All label names
kubectl-metrics labels

# Labels for a specific metric
kubectl-metrics labels --metric container_network_receive_bytes_total
```

### preset

Run a pre-configured named PromQL query. Presets provide quick access to common MTV/Forklift monitoring queries.

Presets marked `[range]` execute a range query over a time window with sensible defaults.
You can override the window with `--start`, `--end`, and `--step`. Instant presets can
also be promoted to range queries by passing `--start`.

```bash
# List presets (shown in --help)
kubectl-metrics preset --help

# Run an instant preset
kubectl-metrics preset --name mtv_migration_status
kubectl-metrics preset --name mtv_migration_pod_rx --namespace mtv-test --format json

# Run a range preset (uses built-in defaults)
kubectl-metrics preset --name mtv_net_throughput_over_time

# Override range defaults
kubectl-metrics preset --name mtv_net_throughput_over_time --start "-2h" --step "30s"

# Group results by namespace
kubectl-metrics preset --name mtv_migration_status --group-by namespace
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | (required) | Preset name |
| `--namespace` | | Namespace filter |
| `--start` | | Start time (overrides preset default for range; promotes instant to range) |
| `--end` | `now` | End time |
| `--step` | | Step interval (overrides preset default) |
| `--format` | `table` | Output format: `table`, `markdown`, `json`, `raw` |
| `--local-time` | `false` | Display timestamps in local timezone instead of UTC |
| `--group-by` | | Label name to split results into sub-tables |

**Available presets:**

| Preset | Type | Description |
|--------|------|-------------|
| `mtv_migration_status` | instant | Migration counts by status (succeeded / failed / running) |
| `mtv_plan_status` | instant | Plan-level status counts |
| `mtv_data_transferred` | instant | Total bytes migrated per plan |
| `mtv_net_throughput` | instant | Migration network throughput |
| `mtv_storage_throughput` | instant | Migration storage throughput |
| `mtv_migration_duration` | instant | Migration duration per plan (seconds) |
| `mtv_migration_pod_rx` | instant | Migration pod receive rate (bytes/sec, top 20) |
| `mtv_migration_pod_tx` | instant | Migration pod transmit rate (bytes/sec, top 20) |
| `mtv_forklift_traffic` | instant | Forklift operator pod network traffic |
| `mtv_namespace_network_rx` | instant | Top 10 namespaces by network receive rate |
| `mtv_namespace_network_tx` | instant | Top 10 namespaces by network transmit rate |
| `mtv_network_errors` | instant | Network errors + drops by namespace (top 10) |
| `mtv_vmi_migrations_pending` | instant | KubeVirt VMI migrations in pending phase |
| `mtv_vmi_migrations_running` | instant | KubeVirt VMI migrations in running phase |
| `mtv_net_throughput_over_time` | range | Migration network throughput trend (default: -1h, step 60s) |
| `mtv_storage_throughput_over_time` | range | Migration storage throughput trend (default: -1h, step 60s) |
| `mtv_data_transferred_over_time` | range | Data transfer progress over time (default: -6h, step 5m) |
| `mtv_migration_status_over_time` | range | Migration status counts over time (default: -6h, step 5m) |
| `mtv_migration_pod_rx_over_time` | range | Migration pod receive rate trend, top 20 (default: -1h, step 60s) |
| `mtv_namespace_network_rx_over_time` | range | Top 10 namespaces by RX rate trend (default: -1h, step 60s) |

## Global Flags

All commands accept standard kubectl flags:

| Flag | Description |
|------|-------------|
| `--kubeconfig` | Path to kubeconfig file |
| `--context` | Kubeconfig context to use |
| `--server` / `-s` | Kubernetes API server URL |
| `--token` | Bearer token for authentication |
| `--namespace` / `-n` | Namespace scope |
| `--url` | Prometheus/Thanos URL override (skips auto-discovery) |
| `--insecure-skip-tls-verify` | Skip TLS certificate verification |

## Prometheus URL Resolution

When `--url` is not provided, the tool auto-discovers the Prometheus/Thanos URL:

1. Queries the OpenShift route API for the `thanos-querier` route in `openshift-monitoring`
2. Falls back to constructing `https://thanos-querier-openshift-monitoring.apps.<cluster-domain>`

## Output Formats

- **table** (default) — Pretty-printed columns with aligned headers, one column per label, human-readable timestamps (UTC by default), and SI-formatted values
- **markdown** — GitHub-compatible Markdown table (same columns as `table`)
- **json** — JSON array of result entries
- **raw** — Full Prometheus API response as JSON

**Table example:**

```
$ kubectl-metrics query --query 'sum(rate(http_requests_total[5m])) by (status)' --name http_rps
METRIC    STATUS  TIMESTAMP            VALUE
http_rps  200     2025-03-02 14:30:05  42.5
http_rps  500     2025-03-02 14:30:05  1.2
```

**Group-by example:**

```
$ kubectl-metrics preset --name mtv_migration_status --group-by namespace
--- namespace: mtv-prod ---
METRIC                STATUS     TIMESTAMP            VALUE
mtv_migration_status  succeeded  2025-03-02 14:30:05  12
mtv_migration_status  running    2025-03-02 14:30:05  3

--- namespace: mtv-test ---
METRIC                STATUS     TIMESTAMP            VALUE
mtv_migration_status  succeeded  2025-03-02 14:30:05  5
mtv_migration_status  failed     2025-03-02 14:30:05  1
```
