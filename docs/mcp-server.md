# MCP Server

kubectl-metrics includes an MCP (Model Context Protocol) server that exposes Prometheus/Thanos metrics to AI assistants.

## Modes

### Stdio (default)

For local AI assistant integration (Claude Desktop, Cursor IDE).

```bash
kubectl-metrics mcp-server
```

### SSE (HTTP)

For network-accessible deployments (OpenShift Lightspeed, remote clients).

```bash
kubectl-metrics mcp-server --sse --port 8080
kubectl-metrics mcp-server --sse --port 8443 --cert-file tls.crt --key-file tls.key
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
| `query` | Instant PromQL query | `query`, `format`, `name`, `local_time`, `group_by` |
| `query_range` | Range query over time window | `query`, `start`, `end`, `step`, `format`, `name`, `local_time`, `group_by` |
| `discover` | List metric names | `keyword`, `group_by_prefix` |
| `labels` | List labels for a metric | `metric` |
| `preset` | Run a named preset query | `name`, `namespace`, `start`, `end`, `step`, `format`, `local_time`, `group_by` |

The optional `name` flag sets a metric name for the first table column — useful for aggregate queries that don't carry a `__name__` label. Set `local_time` to `true` for local-timezone timestamps. Use `group_by` to split results into sub-tables by a label (e.g. `"namespace"`).

Presets marked `[range]` run as range queries with built-in default time windows. Pass `start`/`end`/`step` to override defaults, or pass `start` on an `[instant]` preset to promote it to a range query.

**Examples:**

```json
{"command": "query", "flags": {"query": "up"}}
{"command": "query", "flags": {"query": "sum(rate(http_requests_total[5m])) by (status)", "name": "http_rps"}}
{"command": "discover", "flags": {"keyword": "mtv", "group_by_prefix": true}}
{"command": "preset", "flags": {"name": "mtv_migration_status", "namespace": "mtv-test"}}
{"command": "preset", "flags": {"name": "mtv_net_throughput_over_time"}}
{"command": "preset", "flags": {"name": "mtv_net_throughput_over_time", "start": "-2h", "step": "30s"}}
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
claude mcp add kubectl-metrics kubectl-metrics mcp-server
```

### Cursor IDE

Settings → MCP → Add Server:
- **Name:** kubectl-metrics
- **Command:** kubectl-metrics
- **Args:** mcp-server

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
