# CLI Usage

kubectl metrics provides subcommands for querying Prometheus / Thanos metrics from the command line.

## Commands

### discover

List available Prometheus metric names.

```bash
# List all metrics
kubectl metrics discover

# Filter by keyword
kubectl metrics discover --keyword mtv

# Group by prefix with counts
kubectl metrics discover --keyword network --group-by-prefix
```

### query

Execute an instant PromQL query (returns current values).

```bash
kubectl metrics query --query "up"
kubectl metrics query --query "up" --selector "namespace=prod"
kubectl metrics query --query "sum(rate(http_requests_total[5m])) by (code)" --name http_rps
kubectl metrics query --query "node_memory_MemAvailable_bytes" --output json
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--query` | (required) | PromQL expression |
| `--output` / `-o` | `markdown` | Output format: `table`, `markdown`, `json`, `raw` |
| `--name` | | Metric name for the first table column (useful for aggregate queries that lack `__name__`) |
| `--local-time` | `false` | Display timestamps in local timezone instead of UTC |
| `--group-by` | | Label name to split results into sub-tables (e.g. `namespace`, `pod`) |
| `--no-pivot` | `false` | Disable pivot table layout for range results (show one row per sample instead) |
| `--selector` / `-l` | | Label selector to filter results post-query (e.g. `"namespace=prod,pod=~nginx.*"`). Operators: `=`, `!=`, `=~`, `!~` |

### query-range

Execute one or more range PromQL queries over a time window. Use `--query` multiple times to run several queries in a single call. Each query is labeled with the corresponding `--name` (or auto-generated `q1`, `q2`, ...).

```bash
kubectl metrics query-range --query "rate(http_requests_total[5m])" --start "-1h"
kubectl metrics query-range --query "rate(http_requests_total[5m])" --start "-1h" --selector "code=200"
kubectl metrics query-range --query "node_cpu_seconds_total" --start "-7d" --step "1h" --output json
kubectl metrics query-range --query "sum(rate(http_requests_total[5m])) by (code)" --start "-1h" --name http_rps

# Multi-query: compare CPU and memory in one call
kubectl metrics query-range \
  --query "sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)" \
  --query "sum(container_memory_working_set_bytes) by (namespace)" \
  --name "cpu" --name "mem" --start "-1h"
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--query` | (required) | PromQL expression (repeatable for multi-query) |
| `--name` | auto (`q1`, `q2`, ...) | Display name for each query (repeatable, positionally paired with `--query`) |
| `--start` | `-1h` | Start time: ISO-8601, Unix epoch, or relative (`-1h`, `-7d`) |
| `--end` | `now` | End time (same formats) |
| `--step` | `60s` | Query resolution step |
| `--output` / `-o` | `markdown` | Output format: `table`, `markdown`, `json`, `raw` |
| `--local-time` | `false` | Display timestamps in local timezone instead of UTC |
| `--group-by` | | Label name to split results into sub-tables (e.g. `namespace`, `pod`) |
| `--no-pivot` | `false` | Disable pivot table layout (show one row per sample instead) |
| `--selector` / `-l` | | Label selector to filter results post-query (e.g. `"namespace=prod,pod=~nginx.*"`). Operators: `=`, `!=`, `=~`, `!~` |

### labels

List Prometheus label names, optionally scoped to a metric.

```bash
# All label names
kubectl metrics labels

# Labels for a specific metric
kubectl metrics labels --metric container_network_receive_bytes_total
```

### preset

Run a pre-configured named PromQL query. Presets provide quick access to common cluster
monitoring and MTV/Forklift migration queries.

Every preset works as both an instant query (default) and a range query.
Pass `--start` to get a time-series trend (e.g. `--start "-1h"`).

```bash
# List presets (shown in --help)
kubectl metrics preset --help

# General cluster health
kubectl metrics preset --name cluster_cpu_utilization
kubectl metrics preset --name cluster_pod_status
kubectl metrics preset --name namespace_cpu_usage

# MTV migration presets
kubectl metrics preset --name mtv_migration_status
kubectl metrics preset --name mtv_migration_pod_rx --namespace mtv-test --output json

# Filter preset results by label
kubectl metrics preset --name mtv_migration_pod_rx --selector "pod=~virt-v2v.*"

# Run an instant preset (uses built-in defaults)
kubectl metrics preset --name mtv_net_throughput

# Override range defaults
kubectl metrics preset --name mtv_net_throughput --start "-2h" --step "30s"

# Promote an instant preset to range
kubectl metrics preset --name cluster_cpu_utilization --start "-1h"

# Group results by namespace
kubectl metrics preset --name mtv_migration_status --group-by namespace
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | (required) | Preset name |
| `--namespace` | | Namespace filter |
| `--start` | | Start time: enables range query (e.g. `-1h`, `-7d`) |
| `--end` | `now` | End time |
| `--step` | `60s` | Step interval (e.g. `15s`, `5m`, `1h`) |
| `--output` / `-o` | `markdown` | Output format: `table`, `markdown`, `json`, `raw` |
| `--local-time` | `false` | Display timestamps in local timezone instead of UTC |
| `--group-by` | | Label name to split results into sub-tables |
| `--no-pivot` | `false` | Disable pivot table layout for range results (show one row per sample instead) |
| `--selector` / `-l` | | Label selector to filter results post-query (e.g. `"namespace=prod,pod=~nginx.*"`). Operators: `=`, `!=`, `=~`, `!~` |

**Available presets:**

*General cluster health:*

| Preset | Description |
|--------|-------------|
| `cluster_cpu_utilization` | Cluster CPU utilization percentage |
| `cluster_memory_utilization` | Cluster memory utilization percentage |
| `cluster_pod_status` | Pod counts by phase (Running, Pending, Failed, Succeeded, Unknown) |
| `cluster_node_readiness` | Node readiness status counts |
| `namespace_cpu_usage` | Top 10 namespaces by CPU usage (cores) |
| `namespace_memory_usage` | Top 10 namespaces by memory usage (bytes) |
| `namespace_network_rx` | Top 10 namespaces by network receive rate |
| `namespace_network_tx` | Top 10 namespaces by network transmit rate |
| `namespace_network_errors` | Network errors + drops by namespace (top 10) |
| `pod_restarts_top10` | Top 10 pods by container restart count |

*MTV / Forklift migrations:*

| Preset | Description |
|--------|-------------|
| `mtv_migration_status` | Migration counts by status (succeeded / failed / running) |
| `mtv_plan_status` | Plan-level status counts |
| `mtv_migration_duration` | Migration duration per plan (seconds) |
| `mtv_avg_migration_duration` | Average migration duration (seconds) |
| `mtv_data_transferred` | Total bytes migrated per plan |
| `mtv_net_throughput` | Migration network throughput |
| `mtv_storage_throughput` | Migration storage throughput |
| `mtv_migration_pod_rx` | Migration pod receive rate (bytes/sec, top 20) |
| `mtv_migration_pod_tx` | Migration pod transmit rate (bytes/sec, top 20) |
| `mtv_forklift_traffic` | Forklift operator pod network traffic (bytes/sec) |
| `mtv_vmi_migrations_pending` | KubeVirt VMI migrations in pending phase |
| `mtv_vmi_migrations_running` | KubeVirt VMI migrations in running phase |

## Shell Completion

Tab completion is supported for both `kubectl metrics` and `oc metrics`. The [install script](installation.md#quick-install-linux--macos) sets this up automatically. For manual setup instructions, see [Shell Completion](installation.md#shell-completion).

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

- **markdown** (default) — GitHub-compatible Markdown table with human-readable timestamps (UTC by default) and SI-formatted values. Range queries use a **pivot layout** by default (one column per label combination, one row per timestamp).
- **table** — Pretty-printed columns with aligned headers (same columns as `markdown`)
- **json** — JSON array of result entries
- **raw** — Full Prometheus API response as JSON

**Instant query example:**

```
$ kubectl metrics query --query 'sum(rate(http_requests_total[5m])) by (code)' --name http_rps
METRIC    CODE  TIMESTAMP            VALUE
http_rps  200   2025-03-02 14:30:05  42.5
http_rps  500   2025-03-02 14:30:05  1.2
```

**Range query example (pivot — default):**

```text
$ kubectl metrics query-range --query 'sum(rate(http_requests_total[5m])) by (code)' --start "-1h" --name http_rps
TIMESTAMP            http_rps/200  http_rps/500
2025-03-02 13:30:05  40.1          0.8
2025-03-02 13:31:05  41.3          1
2025-03-02 13:32:05  42.5          1.2
```

**Range query example (--no-pivot):**

```text
$ kubectl metrics query-range --query 'sum(rate(http_requests_total[5m])) by (code)' --start "-1h" --name http_rps --no-pivot
METRIC    CODE  TIMESTAMP            VALUE
http_rps  200   2025-03-02 13:30:05  40.1
http_rps  200   2025-03-02 13:31:05  41.3
http_rps  200   2025-03-02 13:32:05  42.5
---
http_rps  500   2025-03-02 13:30:05  0.8
http_rps  500   2025-03-02 13:31:05  1
http_rps  500   2025-03-02 13:32:05  1.2
```

**Multi-query example (pivot — default):**

```text
$ kubectl metrics query-range \
    --query 'sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)' \
    --query 'sum(container_memory_working_set_bytes) by (namespace)' \
    --name cpu --name mem --start "-1h"
TIMESTAMP            cpu/ns-a  cpu/ns-b  mem/ns-a      mem/ns-b
2025-03-02 13:30:05  2.5       1.3       21.47 G       8.59 G
2025-03-02 13:31:05  2.6       1.4       21.50 G       8.60 G
```

**Multi-query example (--no-pivot):**

```text
$ kubectl metrics query-range \
    --query 'sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)' \
    --query 'sum(container_memory_working_set_bytes) by (namespace)' \
    --name cpu --name mem --start "-1h" --no-pivot
METRIC  NAMESPACE  TIMESTAMP            VALUE
cpu     ns-a       2025-03-02 13:30:05  2.5
cpu     ns-a       2025-03-02 13:31:05  2.6
---
cpu     ns-b       2025-03-02 13:30:05  1.3
cpu     ns-b       2025-03-02 13:31:05  1.4
---
mem     ns-a       2025-03-02 13:30:05  21.47 G
mem     ns-a       2025-03-02 13:31:05  21.50 G
---
mem     ns-b       2025-03-02 13:30:05  8.59 G
mem     ns-b       2025-03-02 13:31:05  8.60 G
```

**Group-by with pivot example:**

```
$ kubectl metrics preset --name mtv_migration_status --group-by namespace
--- namespace: mtv-prod ---
TIMESTAMP            succeeded  running
2025-03-02 14:30:05  12         3

--- namespace: mtv-test ---
TIMESTAMP            succeeded  failed
2025-03-02 14:30:05  5          1
```

**Group-by example (--no-pivot):**

```
$ kubectl metrics preset --name mtv_migration_status --group-by namespace --no-pivot
--- namespace: mtv-prod ---
METRIC                STATUS     TIMESTAMP            VALUE
mtv_migration_status  succeeded  2025-03-02 14:30:05  12
mtv_migration_status  running    2025-03-02 14:30:05  3

--- namespace: mtv-test ---
METRIC                STATUS     TIMESTAMP            VALUE
mtv_migration_status  succeeded  2025-03-02 14:30:05  5
mtv_migration_status  failed     2025-03-02 14:30:05  1
```
