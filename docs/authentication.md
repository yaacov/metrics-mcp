# Authentication

kubectl metrics uses `k8s.io/client-go` for authentication, supporting all standard kubeconfig methods.

## Supported Auth Methods

| Method | How It Works |
|--------|-------------|
| **Bearer token** | Inline token in kubeconfig (e.g. `oc login`) — used directly |
| **Bearer token file** | Token read from file path in kubeconfig |
| **Exec provider** | External command produces a token (e.g. `oc`, `gcloud`, `aws-iam-authenticator`) |
| **Client certificates** | Mutual TLS — works for K8s API; for Prometheus, a service account token is fetched automatically |
| **OIDC / Auth provider** | Handled by client-go auth plugins |

## How It Works

### Standard Flow (bearer token, exec)

1. `client-go` loads the kubeconfig and resolves credentials
2. `rest.TransportFor()` creates an `http.RoundTripper` that injects auth headers
3. The transport is used for both Prometheus URL auto-discovery and metric queries

### Client Certificate Flow

The Thanos querier on OpenShift uses an OAuth proxy that only accepts bearer tokens, not client certificates. When client cert auth is detected:

1. Client cert transport is used to auto-discover the Prometheus URL (K8s API accepts certs)
2. A Kubernetes `TokenRequest` is made for the `prometheus-k8s` service account in `openshift-monitoring`
3. The resulting short-lived (1h) bearer token is used for Prometheus queries

**Fallback order for TokenRequest:**

| Service Account | Namespace |
|----------------|-----------|
| `prometheus-k8s` | `openshift-monitoring` |
| `thanos-querier` | `openshift-monitoring` |

## Prometheus URL Resolution

When `--url` is not provided, the URL is auto-discovered:

1. Query the OpenShift route API for `thanos-querier` in `openshift-monitoring`
2. Fall back to conventional URL: `https://thanos-querier-openshift-monitoring.apps.<cluster-domain>`

## CLI Flag Overrides

Standard kubectl flags override kubeconfig values:

```bash
# Explicit token
kubectl metrics discover --token sha256~xxxxx

# Explicit server + token
kubectl metrics discover --server https://api.cluster.example.com:6443 --token sha256~xxxxx

# Skip auto-discovery, point directly at Prometheus
kubectl metrics discover --url https://prometheus.example.com

# Different kubeconfig or context
kubectl metrics discover --kubeconfig /path/to/config --context my-cluster
```

## HTTP Mode (MCP Server)

In HTTP mode, per-request credentials can be provided via HTTP headers, which take highest priority:

```
Authorization: Bearer <token>
X-Kubernetes-Server: https://api.cluster.example.com:6443
X-Metrics-Server: https://thanos.example.com
```

This allows a single MCP server instance to serve multiple users, each authenticated with their own token (e.g. OpenShift Lightspeed forwarding the logged-in user's token).

## TLS Verification

### Why metrics need different TLS settings

Standard `oc` / `kubectl` commands connect to the **Kubernetes API server**, whose CA certificate is embedded in your kubeconfig — so TLS verification works automatically.

Metrics commands connect to the **Thanos/Prometheus route**, which is served by the OpenShift **ingress controller**. The ingress controller uses a different CA that is not in your kubeconfig or system trust store. This means TLS verification may fail with `x509: certificate signed by unknown authority` even though `oc get pods` works fine.

### Flags

| Flag | Description |
|------|-------------|
| `--certificate-authority` | Path to a custom CA certificate file (PEM) for verifying Prometheus/K8s connections |
| `--insecure-skip-tls-verify` | Skip all TLS certificate verification (insecure, use for development only) |

Both flags work on all subcommands (CLI and `mcp-server`).

### Quick fix: skip verification

For development or lab clusters where you trust the network:

```bash
kubectl metrics discover --insecure-skip-tls-verify
kubectl metrics mcp-server --insecure-skip-tls-verify
```

### Proper fix: use the ingress CA

On OpenShift, the ingress CA is stored in a ConfigMap. Extract it and pass it to kubectl-metrics:

```bash
# Fetch the ingress CA bundle
oc get configmap -n openshift-config-managed \
  default-ingress-cert \
  -o jsonpath='{.data.ca-bundle\.crt}' > /tmp/ingress-ca.crt

# Use it with metrics commands
kubectl metrics discover --certificate-authority /tmp/ingress-ca.crt
kubectl metrics mcp-server --certificate-authority /tmp/ingress-ca.crt
```

### Container / MCP server

When running the MCP server in a container, mount the CA file and set `MCP_KUBE_CA_CERT`:

```bash
podman run --rm -p 8080:8080 \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  -e MCP_KUBE_CA_CERT=/certs/ingress-ca.crt \
  -v /tmp/ingress-ca.crt:/certs/ingress-ca.crt:ro \
  quay.io/yaacov/kubectl-metrics-mcp-server:latest
```

## Debugging Authentication

Use klog verbosity to see which auth method is detected:

```bash
kubectl metrics discover --v=2
```

Output includes:

```
[auth] API server: https://api.cluster.example.com:6443
[auth] Method: client certificate
[auth] Got token via TokenRequest for openshift-monitoring/prometheus-k8s
[auth] Using service account token for Prometheus
```
