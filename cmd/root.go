// Package cmd implements the cobra CLI commands for kubectl-metrics.
package cmd

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yaacov/kubectl-metrics/pkg/connection"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

var (
	configFlags *genericclioptions.ConfigFlags
	metricsURL  string
)

var rootCmd = &cobra.Command{
	Use:   "kubectl-metrics",
	Short: "Query Prometheus / Thanos metrics on OpenShift clusters",
	Long: `Query Prometheus / Thanos metrics on OpenShift clusters.

It provides a CLI and an MCP server, both backed by shared logic.

Authentication:
  Standard kubectl flags (--kubeconfig, --context, --token, --server, etc.)
  --url        Prometheus/Thanos URL override (skips auto-discovery)

Prometheus URL Resolution:
  1. If --url is provided, use it directly
  2. Try to GET the thanos-querier route via the cluster API
  3. Construct conventional URL from API server base domain`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config, err := configFlags.ToRESTConfig()
		if err != nil {
			klog.V(2).Infof("Could not load kubeconfig: %v", err)
			connection.SetDefaultMetricsURL(metricsURL)
			return
		}

		// Propagate kubeconfig's insecure-skip-tls-verify to the connection
		// default so that per-request transports (e.g. bearer tokens from
		// HTTP headers) also skip verification.
		if config.Insecure {
			connection.SetDefaultInsecureSkipTLS(true)
		}
		// Similarly, use the kubeconfig CA when no explicit --certificate-authority
		// was passed on the command line.
		if connection.GetDefaultCACert() == "" && config.TLSClientConfig.CAFile != "" {
			connection.SetDefaultCACert(config.TLSClientConfig.CAFile)
		}

		logAuthMethod(config)
		connection.SetDefaultKubeServer(config.Host)
		connection.SetDefaultMetricsURL(metricsURL)

		if needsBearerToken(config) {
			setupClientCertAuth(config)
		} else {
			rt, err := buildTransport(config)
			if err != nil {
				klog.V(2).Infof("Could not create authenticated transport: %v", err)
				return
			}
			connection.SetDefaultTransport(rt)
		}
	},
}

// needsBearerToken returns true when the REST config uses client certificates
// but has no bearer token — meaning Prometheus/Thanos (which needs bearer
// auth) won't accept the transport as-is.
func needsBearerToken(config *rest.Config) bool {
	hasCert := config.CertData != nil || config.CertFile != ""
	hasToken := config.BearerToken != "" || config.BearerTokenFile != ""
	return hasCert && !hasToken
}

// setupClientCertAuth handles the case where kubeconfig uses client
// certificates. The K8s API accepts certs, but the Thanos OAuth proxy
// requires a bearer token. We use the cert-based client to request a
// service account token via the TokenRequest API.
func setupClientCertAuth(config *rest.Config) {
	kubeRT, err := buildTransport(config)
	if err != nil {
		klog.V(2).Infof("Could not create kube transport: %v", err)
		return
	}

	// Run auto-discovery eagerly with client-cert transport (which has
	// permissions to read routes), before we switch to the SA bearer token.
	if metricsURL == "" {
		discovered := connection.AutoDiscoverPrometheusURL(config.Host, kubeRT)
		if discovered != "" {
			connection.SetDefaultMetricsURL(discovered)
		}
	}

	// Get a bearer token for Prometheus via the TokenRequest API.
	token := requestServiceAccountToken(config)
	if token != "" {
		klog.V(2).Info("[auth] Using service account token for Prometheus")
		rt, err := connection.NewBearerTokenTransport(token)
		if err != nil {
			klog.V(2).Infof("[auth] Could not create bearer token transport: %v", err)
			connection.SetDefaultTransport(kubeRT)
			return
		}
		connection.SetDefaultTransport(rt)
	} else {
		klog.V(2).Info("[auth] WARNING: could not obtain bearer token; Prometheus calls may fail with 401")
		connection.SetDefaultTransport(kubeRT)
	}
}

// requestServiceAccountToken uses the TokenRequest API to create a
// short-lived bearer token from a service account in the cluster.
func requestServiceAccountToken(config *rest.Config) string {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.V(2).Infof("[auth] Could not create clientset: %v", err)
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type candidate struct {
		namespace, sa string
	}
	candidates := []candidate{
		{"openshift-monitoring", "prometheus-k8s"},
		{"openshift-monitoring", "thanos-querier"},
	}

	expiry := int64(2 * 24 * 3600)
	for _, c := range candidates {
		tokenReq := &authv1.TokenRequest{
			Spec: authv1.TokenRequestSpec{
				ExpirationSeconds: &expiry,
			},
		}
		result, err := clientset.CoreV1().ServiceAccounts(c.namespace).
			CreateToken(ctx, c.sa, tokenReq, metav1.CreateOptions{})
		if err != nil {
			klog.V(2).Infof("[auth] TokenRequest for %s/%s failed: %v", c.namespace, c.sa, err)
			continue
		}
		klog.V(2).Infof("[auth] Got token via TokenRequest for %s/%s", c.namespace, c.sa)
		return result.Status.Token
	}

	return ""
}

func logAuthMethod(config *rest.Config) {
	klog.V(2).Infof("[auth] API server: %s", config.Host)
	switch {
	case config.BearerToken != "":
		klog.V(2).Infof("[auth] Method: bearer token (length %d)", len(config.BearerToken))
	case config.BearerTokenFile != "":
		klog.V(2).Infof("[auth] Method: bearer token file (%s)", config.BearerTokenFile)
	case config.ExecProvider != nil:
		klog.V(2).Infof("[auth] Method: exec provider (%s)", config.ExecProvider.Command)
	case config.CertData != nil || config.CertFile != "":
		klog.V(2).Infof("[auth] Method: client certificate")
	case config.Username != "":
		klog.V(2).Infof("[auth] Method: basic auth (user: %s)", config.Username)
	case config.AuthProvider != nil:
		klog.V(2).Infof("[auth] Method: auth provider (%s)", config.AuthProvider.Name)
	default:
		klog.V(2).Info("[auth] WARNING: no authentication credentials found in REST config")
	}
}

// buildTransport creates an http.RoundTripper from the REST config that
// carries the kubeconfig's credentials, with TLS behavior determined by:
//  1. --insecure-skip-tls-verify: skip verification entirely
//  2. --certificate-authority (connection default): use the custom CA
//  3. Otherwise: use the kubeconfig CA or system CAs
func buildTransport(config *rest.Config) (http.RoundTripper, error) {
	promConfig := rest.CopyConfig(config)

	if connection.GetDefaultInsecureSkipTLS() || config.Insecure {
		promConfig.TLSClientConfig.Insecure = true
		promConfig.TLSClientConfig.CAData = nil
		promConfig.TLSClientConfig.CAFile = ""
	} else if ca := connection.GetDefaultCACert(); ca != "" {
		promConfig.TLSClientConfig.CAFile = ca
		promConfig.TLSClientConfig.CAData = nil
	}

	return rest.TransportFor(promConfig)
}

func init() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	configFlags = genericclioptions.NewConfigFlags(true)
	configFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.PersistentFlags().StringVar(&metricsURL, "url", "", "Prometheus/Thanos URL override (skips auto-discovery)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
