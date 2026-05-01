# Running the MCP Server from a Container Image

Run the kubectl-metrics MCP server as a container — locally with Podman/Docker, or on any container runtime — without installing Go or building from source.

## Image

The pre-built image is available at:

```
quay.io/yaacov/kubectl-metrics-mcp-server:latest
```

The image is multi-arch (`linux/amd64` and `linux/arm64`), runs as non-root (UID 1001), and uses a read-only root filesystem.

## Quick Start

### With a bearer token

```bash
podman run --rm -p 8080:8080 \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  quay.io/yaacov/kubectl-metrics-mcp-server:latest
```

### With a known Prometheus URL (skip auto-discovery)

```bash
podman run --rm -p 8080:8080 \
  -e MCP_METRICS_URL=https://thanos-querier-openshift-monitoring.apps.mycluster.example.com \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  quay.io/yaacov/kubectl-metrics-mcp-server:latest
```

### Skip TLS verification (e.g. development clusters)

```bash
podman run --rm -p 8080:8080 \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  -e MCP_KUBE_INSECURE=true \
  quay.io/yaacov/kubectl-metrics-mcp-server:latest
```

### With a custom CA certificate

```bash
podman run --rm -p 8080:8080 \
  -v ./ca.crt:/tls/ca.crt:ro \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  -e MCP_KUBE_CA_CERT=/tls/ca.crt \
  quay.io/yaacov/kubectl-metrics-mcp-server:latest
```

### With TLS (HTTPS)

Mount certificate and key files into the container:

```bash
podman run --rm -p 8443:8443 \
  -v ./tls.crt:/tls/tls.crt:ro \
  -v ./tls.key:/tls/tls.key:ro \
  -e MCP_PORT=8443 \
  -e MCP_CERT_FILE=/tls/tls.crt \
  -e MCP_KEY_FILE=/tls/tls.key \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  quay.io/yaacov/kubectl-metrics-mcp-server:latest
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_HOST` | `0.0.0.0` | Bind address |
| `MCP_PORT` | `8080` | Listen port |
| `MCP_CERT_FILE` | | TLS certificate path (enables HTTPS) |
| `MCP_KEY_FILE` | | TLS private key path (enables HTTPS) |
| `MCP_KUBE_SERVER` | | Kubernetes API server URL |
| `MCP_KUBE_TOKEN` | | Bearer token for K8s and Prometheus auth |
| `MCP_KUBE_INSECURE` | | Set `true` to skip TLS verification for upstream connections |
| `MCP_KUBE_CA_CERT` | | Path to a CA certificate file for TLS verification |
| `MCP_METRICS_URL` | | Prometheus/Thanos URL override (skips auto-discovery) |

## Connecting an AI Client

Once the container is running, the HTTP endpoint is available at:

```
http://localhost:8080/mcp
```

**Claude Desktop** (add to `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "kubectl-metrics": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

**Cursor IDE:**

Settings → MCP → Add Server → Type: `streamableHttp`, URL: `http://localhost:8080/mcp`

## Per-Request Authentication via Headers

In HTTP mode the container also accepts per-request authentication headers. This lets multiple clients share one server, each with their own credentials:

| Header | Description |
|--------|-------------|
| `Authorization: Bearer <token>` | Bearer token for Prometheus auth |
| `X-Kubernetes-Server: <url>` | Kubernetes API server URL |
| `X-Metrics-Server: <url>` | Prometheus/Thanos URL override |

Header values take precedence over environment variables.

## Building the Image

```bash
make vendor                # populate vendor/ (required for container build)
make image-build-amd64     # linux/amd64
make image-build-arm64     # linux/arm64
make image-build-all       # both architectures
```

Push and create a multi-arch manifest:

```bash
make image-push-all        # push arch images + manifest
```

Override the registry and organization:

```bash
make image-build-amd64 IMAGE_REGISTRY=my-registry.example.com IMAGE_ORG=myorg
```

## Docker Compose Example

```yaml
services:
  kubectl-metrics:
    image: quay.io/yaacov/kubectl-metrics-mcp-server:latest
    ports:
      - "8080:8080"
    environment:
      MCP_KUBE_SERVER: https://api.mycluster.example.com:6443
      MCP_KUBE_TOKEN: ${KUBE_TOKEN}
      MCP_KUBE_INSECURE: "true"
```

Run with:

```bash
KUBE_TOKEN="$(oc whoami -t)" docker compose up
```
