# ---- Builder stage (runs on native platform, cross-compiles for target) ----
FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi10/go-toolset:latest AS builder

ARG TARGETARCH=amd64
ARG VERSION=0.0.0-dev

USER root
WORKDIR /build

# Copy go module files first for better layer caching
COPY go.mod go.sum ./
COPY vendor/ vendor/

# Copy source code
COPY main.go ./
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY mcp/ mcp/

# Build kubectl-metrics
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -a \
    -ldflags "-s -w -X github.com/yaacov/kubectl-metrics/pkg/version.Version=${VERSION}" \
    -o kubectl-metrics \
    main.go

# ---- Runtime stage ----
FROM registry.access.redhat.com/ubi10/ubi-minimal:latest

ARG TARGETARCH=amd64

# Copy binary from builder (set execute permissions during copy)
COPY --from=builder --chmod=755 /build/kubectl-metrics /usr/local/bin/kubectl-metrics

# --- Environment variables ---
# HTTP server settings
ENV MCP_HOST="0.0.0.0" \
    MCP_PORT="8080"

# TLS settings (optional - provide paths to enable TLS)
ENV MCP_CERT_FILE="" \
    MCP_KEY_FILE=""

# Kubernetes authentication (optional - override via HTTP headers in HTTP mode)
ENV MCP_KUBE_SERVER="" \
    MCP_KUBE_TOKEN="" \
    MCP_KUBE_INSECURE="" \
    MCP_KUBE_CA_CERT=""

# Prometheus/Thanos URL override (optional - auto-discovered from cluster if empty)
ENV MCP_METRICS_URL=""

USER 1001
WORKDIR /home/metrics

EXPOSE 8080

ENTRYPOINT ["/bin/sh", "-c", "\
  exec kubectl-metrics mcp-server --http \
    --host \"${MCP_HOST}\" \
    --port \"${MCP_PORT}\" \
    ${MCP_CERT_FILE:+--cert-file \"${MCP_CERT_FILE}\"} \
    ${MCP_KEY_FILE:+--key-file \"${MCP_KEY_FILE}\"} \
    ${MCP_KUBE_SERVER:+--server \"${MCP_KUBE_SERVER}\"} \
    ${MCP_KUBE_TOKEN:+--token \"${MCP_KUBE_TOKEN}\"} \
    $([ \"${MCP_KUBE_INSECURE}\" = \"true\" ] && echo --insecure-skip-tls-verify) \
    ${MCP_KUBE_CA_CERT:+--certificate-authority \"${MCP_KUBE_CA_CERT}\"} \
    ${MCP_METRICS_URL:+--url \"${MCP_METRICS_URL}\"}"]

LABEL name="kubectl-metrics-mcp-server" \
      summary="kubectl-metrics MCP server for AI-assisted Prometheus/Thanos monitoring" \
      description="MCP (Model Context Protocol) server exposing Prometheus/Thanos metrics queries for AI assistants. Runs in HTTP mode using Streamable HTTP transport." \
      io.k8s.display-name="kubectl-metrics MCP Server" \
      io.k8s.description="MCP server for kubectl-metrics providing AI-assisted Prometheus/Thanos metric queries via Streamable HTTP transport." \
      maintainer="Yaacov Zamir <kobi.zamir@gmail.com>"
