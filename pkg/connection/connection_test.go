package connection

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInsecureTransport(t *testing.T) {
	rt := InsecureTransport()
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("InsecureTransport did not return *http.Transport")
	}
	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureTransport should set InsecureSkipVerify=true")
	}
}

func TestNewCATransport_ValidCert(t *testing.T) {
	caFile := writeTempCA(t)

	rt, err := NewCATransport(caFile)
	if err != nil {
		t.Fatalf("NewCATransport failed: %v", err)
	}

	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("NewCATransport did not return *http.Transport")
	}
	if tr.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig is nil")
	}
	if tr.TLSClientConfig.RootCAs == nil {
		t.Error("RootCAs should be set when CA file is provided")
	}
	if tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be false when using CA")
	}
}

func TestNewCATransport_MissingFile(t *testing.T) {
	_, err := NewCATransport("/nonexistent/ca.crt")
	if err == nil {
		t.Error("NewCATransport should fail for missing file")
	}
}

func TestNewCATransport_InvalidPEM(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.crt")
	if err := os.WriteFile(tmp, []byte("not a certificate"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewCATransport(tmp)
	if err == nil {
		t.Error("NewCATransport should fail for invalid PEM")
	}
}

func TestDefaultTransportBase_Insecure(t *testing.T) {
	defer resetDefaults()

	SetDefaultInsecureSkipTLS(true)
	SetDefaultCACert("")

	rt, err := DefaultTransportBase()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("should be insecure when defaultInsecureSkipTLS is true")
	}
}

func TestDefaultTransportBase_WithCA(t *testing.T) {
	defer resetDefaults()

	caFile := writeTempCA(t)
	SetDefaultInsecureSkipTLS(false)
	SetDefaultCACert(caFile)

	rt, err := DefaultTransportBase()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("should not be insecure when CA is set")
	}
	if tr.TLSClientConfig.RootCAs == nil {
		t.Error("RootCAs should be set")
	}
}

func TestDefaultTransportBase_SystemCAs(t *testing.T) {
	defer resetDefaults()

	SetDefaultInsecureSkipTLS(false)
	SetDefaultCACert("")

	rt, err := DefaultTransportBase()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("should not be insecure")
	}
}

func TestDefaultTransportBase_InsecureOverridesCA(t *testing.T) {
	defer resetDefaults()

	caFile := writeTempCA(t)
	SetDefaultInsecureSkipTLS(true)
	SetDefaultCACert(caFile)

	rt, err := DefaultTransportBase()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("insecure flag should take priority over CA cert")
	}
}

func TestDefaultTransportBase_InvalidCA_ReturnsError(t *testing.T) {
	defer resetDefaults()

	SetDefaultInsecureSkipTLS(false)
	SetDefaultCACert("/nonexistent/ca.crt")

	_, err := DefaultTransportBase()
	if err == nil {
		t.Error("DefaultTransportBase should return error for invalid CA path")
	}
}

func TestNewBearerTokenTransport_UsesDefaultBase(t *testing.T) {
	defer resetDefaults()

	SetDefaultInsecureSkipTLS(true)

	rt, err := NewBearerTokenTransport("test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bt, ok := rt.(*bearerTokenTransport)
	if !ok {
		t.Fatal("expected *bearerTokenTransport")
	}
	if bt.token != "test-token" {
		t.Errorf("token = %q, want %q", bt.token, "test-token")
	}

	base, ok := bt.base.(*http.Transport)
	if !ok {
		t.Fatal("base should be *http.Transport")
	}
	if !base.TLSClientConfig.InsecureSkipVerify {
		t.Error("base transport should be insecure when flag is set")
	}
}

func TestNewBearerTokenTransport_InvalidCA_ReturnsError(t *testing.T) {
	defer resetDefaults()

	SetDefaultInsecureSkipTLS(false)
	SetDefaultCACert("/nonexistent/ca.crt")

	_, err := NewBearerTokenTransport("test-token")
	if err == nil {
		t.Error("NewBearerTokenTransport should return error for invalid CA path")
	}
}

func TestSetGetDefaults(t *testing.T) {
	defer resetDefaults()

	SetDefaultCACert("/some/path.crt")
	if got := GetDefaultCACert(); got != "/some/path.crt" {
		t.Errorf("GetDefaultCACert() = %q, want %q", got, "/some/path.crt")
	}

	SetDefaultInsecureSkipTLS(true)
	if !GetDefaultInsecureSkipTLS() {
		t.Error("GetDefaultInsecureSkipTLS() should be true")
	}

	SetDefaultInsecureSkipTLS(false)
	if GetDefaultInsecureSkipTLS() {
		t.Error("GetDefaultInsecureSkipTLS() should be false")
	}
}

func resetDefaults() {
	defaultTransport = nil
	defaultKubeServer = ""
	defaultMetricsURL = ""
	defaultCACertPath = ""
	defaultInsecureSkipTLS = false
}

func writeTempCA(t *testing.T) string {
	t.Helper()

	certPEM := generateSelfSignedCert(t)
	tmp := filepath.Join(t.TempDir(), "ca.crt")
	if err := os.WriteFile(tmp, certPEM, 0644); err != nil {
		t.Fatal(err)
	}
	return tmp
}

func generateSelfSignedCert(t *testing.T) []byte {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-ca"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
}
