# kubectl-metrics

Query Prometheus / Thanos metrics on OpenShift clusters — as a CLI and an MCP server for AI assistants.

## Installation

Download or build the `kubectl-metrics` binary, then place it anywhere on your `PATH`:

```bash
# Build from source
make build

# Copy the binary to a directory in your PATH
sudo cp kubectl-metrics /usr/local/bin/
```

Once `kubectl-metrics` is on your `PATH`, kubectl automatically discovers it as a plugin and you can run it as `kubectl metrics`:

```bash
kubectl metrics --help
```

## Quick Start

```bash
# CLI usage (auto-discovers Prometheus from your kubeconfig)
kubectl metrics discover
kubectl metrics discover --keyword mtv
kubectl metrics query --query "up"
kubectl metrics query --query "sum(rate(http_requests_total[5m])) by (code)" --name http_rps
kubectl metrics query-range --query "rate(http_requests_total[5m])" --start "-1h"
kubectl metrics query-range \
  --query "sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)" \
  --query "sum(container_memory_working_set_bytes) by (namespace)" \
  --name cpu --name mem --start "-1h"
kubectl metrics preset --name mtv_migration_status

# MCP server (stdio, for Claude Desktop / Cursor IDE)
kubectl metrics mcp-server

# MCP server (SSE, for OpenShift Lightspeed)
kubectl metrics mcp-server --sse --port 8080

# MCP server from container image
podman run --rm -p 8080:8080 \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  quay.io/yaacov/kubectl-metrics-mcp-server:latest
```

## AI Assistant Setup

**Claude Desktop:**

```bash
claude mcp add kubectl-metrics kubectl metrics mcp-server
```

**Cursor IDE:**

Settings → MCP → Add Server → Name: `kubectl-metrics`, Command: `kubectl`, Args: `metrics mcp-server`

## Authentication

Uses standard kubectl flags (`--kubeconfig`, `--context`, `--token`, `--server`). Auto-discovers the Prometheus/Thanos URL from the cluster's thanos-querier route.

For client certificate auth (e.g. Kind/minikube), automatically requests a service account token via the Kubernetes TokenRequest API.

## Deploy on OpenShift

```bash
make deploy              # Deployment + Service
make deploy-route        # External Route with TLS
make deploy-olsconfig    # Register with OpenShift Lightspeed
```

## Similar Tools

Several other tools exist for querying Prometheus from the command line.
**promtool** (official Prometheus CLI) and **promql-cli** offer standalone PromQL querying,
**prometheus-mcp-server** exposes Prometheus to AI assistants via MCP, and
**kubectl-prom** is a lightweight kubectl plugin for in-cluster queries.

kubectl-metrics combines all three angles — kubectl-native CLI, OpenShift/Thanos auto-discovery with preset queries, and a built-in MCP server — in a single binary.
See [Similar Tools](docs/similar-tools.md) for a detailed comparison.

## Documentation

See the [docs/](docs/) directory for detailed guides:

- [CLI Usage](docs/cli-usage.md)
- [MCP Server](docs/mcp-server.md)
- [Containerized](docs/containerized.md)
- [Authentication](docs/authentication.md)
- [Deployment](docs/deployment.md)
- [Similar Tools](docs/similar-tools.md)

## Development

```bash
make help          # Show all targets
make build         # Build binary
make test          # Run tests
make fmt           # Format code
make lint          # Run linters (requires: make install-tools)
make vendor        # Populate vendor/
make image-build-amd64   # Build container image
```

## License

Apache License 2.0
