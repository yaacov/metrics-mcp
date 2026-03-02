// Package connection provides credential resolution and Prometheus endpoint
// discovery for kubectl-metrics.
//
// Credentials flow through three tiers (highest priority first):
//  1. SSE HTTP headers (per-session)
//  2. CLI defaults (kubeconfig via client-go)
//  3. Auto-discovered from cluster
package connection

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	transportKey  contextKey = "transport"
	kubeServerKey contextKey = "kubernetes_server"
	metricsURLKey contextKey = "metrics_server"
)

// --- Transport helpers ---

// bearerTokenTransport injects a static bearer token into every request.
// Used for MCP/SSE sessions where the token arrives via HTTP headers.
type bearerTokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(r)
}

// InsecureTransport returns a transport that skips TLS verification.
func InsecureTransport() http.RoundTripper {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}

// NewBearerTokenTransport creates a transport that adds a bearer token
// to every request, with InsecureSkipVerify for self-signed certs.
func NewBearerTokenTransport(token string) http.RoundTripper {
	return &bearerTokenTransport{
		token: token,
		base:  InsecureTransport(),
	}
}

// --- Context accessors ---

// WithTransport adds an authenticated transport to the context.
func WithTransport(ctx context.Context, rt http.RoundTripper) context.Context {
	return context.WithValue(ctx, transportKey, rt)
}

// GetTransport retrieves the authenticated transport from the context.
func GetTransport(ctx context.Context) (http.RoundTripper, bool) {
	if ctx == nil {
		return nil, false
	}
	rt, ok := ctx.Value(transportKey).(http.RoundTripper)
	return rt, ok
}

// WithKubeServer adds a Kubernetes API server URL to the context.
func WithKubeServer(ctx context.Context, server string) context.Context {
	return context.WithValue(ctx, kubeServerKey, server)
}

// GetKubeServer retrieves the Kubernetes API server URL from the context.
func GetKubeServer(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	server, ok := ctx.Value(kubeServerKey).(string)
	return server, ok
}

// WithMetricsURL adds a Prometheus/Thanos URL to the context.
func WithMetricsURL(ctx context.Context, metricsURL string) context.Context {
	return context.WithValue(ctx, metricsURLKey, metricsURL)
}

// GetMetricsURL retrieves the Prometheus/Thanos URL from the context.
func GetMetricsURL(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	u, ok := ctx.Value(metricsURLKey).(string)
	return u, ok
}

// WithCredsFromHeaders extracts credentials from HTTP headers and adds them
// to the context. A bearer token is wrapped into a transport.
//
// Supported headers:
//   - Authorization: Bearer <token>
//   - X-Kubernetes-Server: <url>
//   - X-Metrics-Server: <url>
func WithCredsFromHeaders(ctx context.Context, headers http.Header) context.Context {
	if headers == nil {
		return ctx
	}

	if auth := headers.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != "" {
			ctx = WithTransport(ctx, NewBearerTokenTransport(token))
		}
	}

	if server := headers.Get("X-Kubernetes-Server"); server != "" {
		ctx = WithKubeServer(ctx, server)
	}

	if metricsURL := headers.Get("X-Metrics-Server"); metricsURL != "" {
		ctx = WithMetricsURL(ctx, metricsURL)
	}

	return ctx
}

// --- Package-level defaults (set from CLI flags at startup) ---

var (
	defaultTransport  http.RoundTripper
	defaultKubeServer string
	defaultMetricsURL string
)

// SetDefaultTransport sets the default authenticated transport from kubeconfig.
func SetDefaultTransport(rt http.RoundTripper) { defaultTransport = rt }

// SetDefaultKubeServer sets the default Kubernetes API server URL.
func SetDefaultKubeServer(s string) { defaultKubeServer = s }

// SetDefaultMetricsURL sets the default Prometheus/Thanos URL from CLI flags.
func SetDefaultMetricsURL(u string) { defaultMetricsURL = u }

// ResolveConnection returns (prometheusURL, transport) using the 3-tier precedence:
//
//	context (SSE headers) > CLI defaults > auto-discovery
func ResolveConnection(ctx context.Context) (string, http.RoundTripper) {
	// 1. Context (from SSE headers)
	metricsURL, _ := GetMetricsURL(ctx)
	rt, _ := GetTransport(ctx)
	kubeServer, _ := GetKubeServer(ctx)

	// 2. Fall back to CLI defaults
	if metricsURL == "" {
		metricsURL = defaultMetricsURL
	}
	if rt == nil {
		rt = defaultTransport
	}
	if kubeServer == "" {
		kubeServer = defaultKubeServer
	}

	// 3. If no transport available, use a bare insecure transport
	if rt == nil {
		rt = InsecureTransport()
	}

	// 4. Auto-discover Prometheus URL if still empty
	if metricsURL == "" {
		metricsURL = AutoDiscoverPrometheusURL(kubeServer, rt)
	}

	return metricsURL, rt
}

// AutoDiscoverPrometheusURL attempts to find the Prometheus/Thanos URL from the cluster.
//
// Strategy:
//  1. Try to GET the thanos-querier route in openshift-monitoring via the cluster API.
//  2. If that fails, construct a conventional URL from the API server base domain.
func AutoDiscoverPrometheusURL(kubeServer string, rt http.RoundTripper) string {
	if kubeServer == "" {
		return ""
	}

	routeURL := strings.TrimRight(kubeServer, "/") +
		"/apis/route.openshift.io/v1/namespaces/openshift-monitoring/routes/thanos-querier"

	discovered := tryGetRouteHost(routeURL, rt)
	if discovered != "" {
		klog.V(1).Infof("Auto-discovered Prometheus URL from route: %s", discovered)
		return discovered
	}

	// Fall back to conventional URL construction
	domain := extractAppsDomain(kubeServer)
	if domain != "" {
		conventional := "https://thanos-querier-openshift-monitoring." + domain
		klog.V(1).Infof("Using conventional Prometheus URL: %s", conventional)
		return conventional
	}

	return ""
}

// tryGetRouteHost queries the OpenShift route API and returns the host if found.
func tryGetRouteHost(routeURL string, rt http.RoundTripper) string {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: rt,
	}

	req, err := http.NewRequest(http.MethodGet, routeURL, nil)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		klog.V(2).Infof("Route discovery failed: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.V(2).Infof("Route discovery returned HTTP %d", resp.StatusCode)
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var route struct {
		Spec struct {
			Host string `json:"host"`
			TLS  *struct {
				Termination string `json:"termination"`
			} `json:"tls"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(body, &route); err != nil {
		return ""
	}

	host := route.Spec.Host
	if host == "" {
		return ""
	}

	scheme := "https"
	if route.Spec.TLS == nil {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}

// extractAppsDomain extracts the apps.<cluster-domain> from a Kubernetes API server URL.
// For example, https://api.mycluster.example.com:6443 → apps.mycluster.example.com
func extractAppsDomain(kubeServer string) string {
	u, err := url.Parse(kubeServer)
	if err != nil {
		return ""
	}

	host := u.Hostname()
	if strings.HasPrefix(host, "api.") {
		return "apps." + strings.TrimPrefix(host, "api.")
	}

	return ""
}
