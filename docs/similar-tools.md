# Similar Tools

Several open-source tools let you query Prometheus metrics from the command line or
expose them to AI assistants. This page compares them with kubectl-metrics so you
can pick the right tool for your workflow.

## Overview

| Feature | kubectl-metrics | promtool | promql-cli | prometheus-mcp-server | kubectl-prom |
|---|---|---|---|---|---|
| Instant queries | Yes | Yes | Yes | Yes | Yes |
| Range queries | Yes | Yes | Yes | Yes | No |
| Metric discovery | Yes | No | Yes | Yes | No |
| Label listing | Yes | No | Yes | No | No |
| Preset queries | 20 MTV presets (14 instant + 6 range) | No | No | No | No |
| PromQL reference | Built-in | No | No | No | No |
| kubectl plugin | Yes | No | No | No | Yes |
| kubeconfig auth | Yes | No | No | No | Yes |
| OpenShift auto-discovery | Yes | No | No | No | No |
| MCP server (AI) | Yes (stdio + SSE) | No | No | Yes (stdio) | No |
| Output formats | table / markdown / json / raw / csv / tsv | json | table / json / csv | json | table |
| Language | Go | Go | Go | Python / TypeScript | Go |
| License | Apache-2.0 | Apache-2.0 | Apache-2.0 | MIT | MIT |

## Tool-by-Tool Comparison

### promtool

[promtool](https://prometheus.io/docs/prometheus/latest/command-line/promtool/) is the
official Prometheus CLI. Its `query instant` and `query range` subcommands accept a
Prometheus server URL and a PromQL expression.

**Strengths:**
- Ships with every Prometheus release — no extra install.
- Includes rule validation, TSDB inspection, and unit-test commands beyond querying.
- Trusted, well-documented, and maintained by the Prometheus project.

**Limitations:**
- Requires a direct Prometheus URL; no kubeconfig or service discovery.
- No metric discovery, label listing, or preset queries.
- No MCP server or AI integration.
- JSON-only output; no human-friendly tables.

### promql-cli

[promql-cli](https://github.com/nalbury/promql-cli) is a community Go tool focused
on a friendly query experience with ASCII graphs for range results.

**Strengths:**
- Clean UX with table, JSON, and CSV output.
- Built-in metric and label discovery (`metrics`, `labels`, `meta` commands).
- ASCII graph rendering for range queries.
- YAML config file for persistent settings.

**Limitations:**
- Requires a direct Prometheus URL; no Kubernetes or OpenShift awareness.
- No preset queries or built-in PromQL help.
- No MCP server or AI integration.
- No bearer-token or client-cert auth flows for in-cluster Prometheus.

### prometheus-mcp-server

[prometheus-mcp-server](https://github.com/pab1it0/prometheus-mcp-server) is a Python /
TypeScript MCP server that exposes Prometheus to AI assistants such as Claude Desktop,
Cursor, and VS Code Copilot.

**Strengths:**
- Purpose-built for AI assistant integration.
- Provides metric metadata and target info alongside queries.
- Available as a Docker image and a pip package.
- Supports basic auth and bearer token.

**Limitations:**
- MCP only — no standalone CLI for human use.
- Requires a direct Prometheus URL; no kubeconfig, no OpenShift route discovery.
- No preset queries, no PromQL reference.
- No SSE mode for multi-user or Lightspeed-style deployments.
- Does not integrate with kubectl workflows.

### kubectl-prom

[kubectl-prom](https://github.com/pete911/kubectl-prom) is a lightweight kubectl
plugin that forwards PromQL queries through the Kubernetes API server.

**Strengths:**
- True kubectl plugin using kubeconfig for authentication.
- Simple and focused — easy to install and run.

**Limitations:**
- Supports only instant queries; no range queries.
- No metric discovery, label listing, or preset queries.
- No MCP server or AI integration.
- No OpenShift Thanos/route auto-discovery.

## Where kubectl-metrics Fits

kubectl-metrics was designed for operators and AI assistants working with
OpenShift and KubeVirt/MTV clusters. It combines three capabilities that are
otherwise spread across separate tools:

1. **kubectl-native CLI** — Uses kubeconfig, context, and namespace flags just like
   kubectl. Auto-discovers the Thanos route on OpenShift so you never need to look
   up a Prometheus URL.

2. **Domain-specific presets** — Ships with 20 preset queries for MTV / Forklift
   migration monitoring (status, throughput, duration, network traffic), including
   6 range presets for time-series trends. These give immediate answers without
   writing PromQL.

3. **Built-in MCP server** — Runs in stdio mode for local AI tools (Claude Desktop,
   Cursor) or in SSE mode with per-session auth for multi-user deployments
   (OpenShift Lightspeed). No separate MCP server process is needed.

If you only need basic PromQL from a laptop against a known Prometheus URL,
promtool or promql-cli may be simpler. If you need only MCP without a CLI,
prometheus-mcp-server is a lighter option. kubectl-metrics is the right choice when
you want all three — CLI, Kubernetes integration, and AI — in one binary.
