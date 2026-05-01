// Package connection provides credential resolution and Prometheus endpoint
// discovery for kubectl-metrics.
//
// Credentials flow through three tiers (highest priority first):
//  1. HTTP headers (per-request)
//  2. CLI defaults (kubeconfig via client-go)
//  3. Auto-discovered from cluster
package connection

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
// Used for MCP HTTP sessions where the token arrives via HTTP headers.
type bearerTokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(r)
}

// cloneDefaultTransport returns a clone of http.DefaultTransport with all its
// proxy, pooling, timeout, and HTTP/2 settings preserved.
func cloneDefaultTransport() *http.Transport {
	return http.DefaultTransport.(*http.Transport).Clone()
}

// InsecureTransport returns a transport that skips TLS verification.
func InsecureTransport() http.RoundTripper {
	tr := cloneDefaultTransport()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return tr
}

// NewCATransport creates a transport that verifies TLS using a custom CA
// certificate file. Returns an error if the file cannot be read or parsed.
func NewCATransport(caFile string) (http.RoundTripper, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("reading CA certificate %s: %w", caFile, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("no valid certificates found in %s", caFile)
	}

	tr := cloneDefaultTransport()
	tr.TLSClientConfig = &tls.Config{RootCAs: pool}
	return tr, nil
}

// DefaultTransportBase returns the appropriate base transport based on the
// configured TLS mode. Returns an error if a custom CA path is configured
// but cannot be loaded.
func DefaultTransportBase() (http.RoundTripper, error) {
	if defaultInsecureSkipTLS {
		return InsecureTransport(), nil
	}
	if defaultCACertPath != "" {
		rt, err := NewCATransport(defaultCACertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load custom CA at %s: %w", defaultCACertPath, err)
		}
		return rt, nil
	}
	return cloneDefaultTransport(), nil
}

// NewBearerTokenTransport creates a transport that adds a bearer token
// to every request, using the configured TLS mode (insecure, custom CA,
// or system CAs). Returns an error if a custom CA is configured but invalid.
func NewBearerTokenTransport(token string) (http.RoundTripper, error) {
	base, err := DefaultTransportBase()
	if err != nil {
		return nil, err
	}
	return &bearerTokenTransport{
		token: token,
		base:  base,
	}, nil
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
func WithCredsFromHeaders(ctx context.Context, headers http.Header) (context.Context, error) {
	if headers == nil {
		return ctx, nil
	}

	if auth := headers.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != "" {
			rt, err := NewBearerTokenTransport(token)
			if err != nil {
				return ctx, err
			}
			ctx = WithTransport(ctx, rt)
		}
	}

	if server := headers.Get("X-Kubernetes-Server"); server != "" {
		ctx = WithKubeServer(ctx, server)
	}

	if metricsURL := headers.Get("X-Metrics-Server"); metricsURL != "" {
		ctx = WithMetricsURL(ctx, metricsURL)
	}

	return ctx, nil
}

// --- Package-level defaults (set from CLI flags at startup) ---

var (
	defaultTransport       http.RoundTripper
	defaultKubeServer      string
	defaultMetricsURL      string
	defaultCACertPath      string
	defaultInsecureSkipTLS bool
)

// SetDefaultTransport sets the default authenticated transport from kubeconfig.
func SetDefaultTransport(rt http.RoundTripper) { defaultTransport = rt }

// SetDefaultKubeServer sets the default Kubernetes API server URL.
func SetDefaultKubeServer(s string) { defaultKubeServer = s }

// SetDefaultMetricsURL sets the default Prometheus/Thanos URL from CLI flags.
func SetDefaultMetricsURL(u string) { defaultMetricsURL = u }

// SetDefaultCACert sets the path to a custom CA certificate for TLS verification.
func SetDefaultCACert(path string) { defaultCACertPath = path }

// GetDefaultCACert returns the configured CA certificate path.
func GetDefaultCACert() string { return defaultCACertPath }

// SetDefaultInsecureSkipTLS sets whether to skip TLS verification.
func SetDefaultInsecureSkipTLS(skip bool) { defaultInsecureSkipTLS = skip }

// GetDefaultInsecureSkipTLS returns whether TLS verification is skipped.
func GetDefaultInsecureSkipTLS() bool { return defaultInsecureSkipTLS }

// ResolveConnection returns (prometheusURL, transport, error) using the 3-tier precedence:
//
//	context (HTTP headers) > CLI defaults > auto-discovery
func ResolveConnection(ctx context.Context) (string, http.RoundTripper, error) {
	// 1. Context (from HTTP headers)
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

	// 3. If no transport available, use the configured TLS mode
	if rt == nil {
		var err error
		rt, err = DefaultTransportBase()
		if err != nil {
			return "", nil, err
		}
	}

	// 4. Auto-discover Prometheus URL if still empty
	if metricsURL == "" {
		metricsURL = AutoDiscoverPrometheusURL(kubeServer, rt)
	}

	return metricsURL, rt, nil
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
