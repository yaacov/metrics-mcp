# MCP Server

kubectl metrics includes an MCP (Model Context Protocol) server that exposes Prometheus/Thanos metrics to AI assistants.

## Modes

### Stdio (default)

For local AI assistant integration (Claude Desktop, Cursor IDE).

```bash
kubectl metrics mcp-server
```

### SSE (HTTP)

For network-accessible deployments (OpenShift Lightspeed, remote clients).

```bash
kubectl metrics mcp-server --sse --port 8080
kubectl metrics mcp-server --sse --port 8443 --cert-file tls.crt --key-file tls.key
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--sse` | `false` | Enable SSE mode over HTTP |
| `--port` | `9091` | Listen port |
| `--host` | `0.0.0.0` | Bind address |
| `--cert-file` | | TLS certificate (enables HTTPS) |
| `--key-file` | | TLS private key (enables HTTPS) |

## MCP Tools

The server exposes two tools:

### metrics_read

Query Prometheus/Thanos metrics. Subcommands:

| Command | Description | Key Flags |
|---------|-------------|-----------|
| `query` | Instant PromQL query | `query`, `output`, `file`, `name`, `local_time`, `group_by`, `no_pivot`, `no_headers`, `selector` |
| `query_range` | Range query over time window (supports multi-query) | `query`, `name`, `start`, `end`, `step`, `output`, `file`, `local_time`, `group_by`, `no_pivot`, `no_headers`, `selector` |
| `discover` | List metric names | `keyword`, `group_by_prefix` |
| `labels` | List labels for a metric | `metric` |
| `preset` | Run a named preset query | `name`, `namespace`, `start`, `end`, `step`, `output`, `file`, `local_time`, `group_by`, `no_pivot`, `no_headers`, `selector` |

For `query_range`, the `query` and `name` flags accept either a string or an array of strings for multi-query support. Each `query[i]` is labeled with the corresponding `name[i]`. If `name` has fewer entries than `query`, the missing names are auto-generated as `q1`, `q2`, ... (e.g. three queries with one name `["cpu"]` produces labels `cpu`, `q2`, `q3`). Extra `name` entries beyond the number of queries are ignored. All queries share the same `start`, `end`, and `step`.

Set `local_time` to `true` for local-timezone timestamps. Use `group_by` to split results into sub-tables by a label (e.g. `"namespace"` or `"__name__"` to split by query). Use `selector` to filter results by labels post-query (e.g. `"namespace=prod,pod=~nginx.*"`); supported operators: `=` (equal), `!=` (not equal), `=~` (regex), `!~` (negative regex).

Range queries default to a pivot table layout (one column per name/label combination, one row per timestamp). Set `no_pivot: true` to use the traditional row-per-sample layout.

Pass `file` to write the formatted output directly to a file and return only a short summary (row count and column names). The path must be under the system temp directory (e.g. `/tmp/data.tsv`). This is useful for large range-query results intended for gnuplot or other external tools, as it avoids returning large text payloads.

Every preset works as both an instant query (default) and a range query. Pass `start` to get a time-series trend (e.g. `start: "-1h"`).

**Examples:**

```json
{"command": "query", "flags": {"query": "up"}}
{"command": "query", "flags": {"query": "sum(rate(http_requests_total[5m])) by (code)", "name": "http_rps"}}
{"command": "query_range", "flags": {"query": "rate(http_requests_total[5m])", "start": "-1h"}}
{"command": "query_range", "flags": {"query": ["sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)", "sum(container_memory_working_set_bytes) by (namespace)"], "name": ["cpu", "mem"], "start": "-1h"}}
{"command": "discover", "flags": {"keyword": "mtv", "group_by_prefix": true}}
{"command": "preset", "flags": {"name": "mtv_migration_status", "namespace": "mtv-test"}}
{"command": "preset", "flags": {"name": "mtv_net_throughput"}}
{"command": "preset", "flags": {"name": "mtv_net_throughput", "start": "-2h", "step": "30s"}}
{"command": "query_range", "flags": {"query": "rate(http_requests_total[5m])", "start": "-1h", "output": "tsv", "file": "/tmp/data.tsv"}}
{"command": "query", "flags": {"query": "up", "selector": "namespace=prod,job=~prom.*"}}
```

### metrics_help

Get help for subcommands and PromQL syntax.

```json
{"command": "query"}
{"command": "promql"}
```

## AI Assistant Setup

### Claude Desktop

```bash
claude mcp add kubectl-metrics kubectl metrics mcp-server
```

### Cursor IDE

Settings → MCP → Add Server:
- **Name:** kubectl-metrics
- **Command:** kubectl
- **Args:** metrics mcp-server

### SSE Mode (remote)

Point the client at the SSE endpoint:

```
http://<host>:<port>/sse
```

## SSE Authentication

In SSE mode, per-session credentials are passed via HTTP headers:

| Header | Description |
|--------|-------------|
| `Authorization: Bearer <token>` | Bearer token for Prometheus auth |
| `X-Kubernetes-Server: <url>` | Kubernetes API server URL |
| `X-Metrics-Server: <url>` | Prometheus/Thanos URL override |

**Precedence:** HTTP headers (per-session) > CLI flags > kubeconfig / auto-discovery
