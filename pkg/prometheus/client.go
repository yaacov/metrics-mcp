// Package prometheus provides an HTTP client for the Prometheus / Thanos query API.
package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// relativeRE matches relative time offsets like -7d, -1h, -30m, -2w.
var relativeRE = regexp.MustCompile(`^-(\d+)([smhdw])$`)

// relativeUnits maps unit chars to their duration multipliers.
var relativeUnits = map[byte]time.Duration{
	's': time.Second,
	'm': time.Minute,
	'h': time.Hour,
	'd': 24 * time.Hour,
	'w': 7 * 24 * time.Hour,
}

// ResolveTime converts a time string to an ISO-8601 timestamp.
// Accepts ISO-8601, Unix epoch seconds, "now", or relative offsets
// like -7d, -1h, -30m, -2w.  Returns defaultTime formatted as
// ISO-8601 when value is empty.
func ResolveTime(value string, defaultTime time.Time) string {
	if value == "" {
		return defaultTime.UTC().Format(time.RFC3339)
	}

	stripped := strings.TrimSpace(strings.ToLower(value))

	if stripped == "now" {
		return time.Now().UTC().Format(time.RFC3339)
	}

	if m := relativeRE.FindStringSubmatch(stripped); m != nil {
		amount, _ := strconv.Atoi(m[1])
		unit := m[2][0]
		delta := time.Duration(amount) * relativeUnits[unit]
		return time.Now().UTC().Add(-delta).Format(time.RFC3339)
	}

	return value
}

// Client is a thin wrapper around the Prometheus HTTP API.
// Authentication is handled by the http.RoundTripper provided at construction.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Prometheus API client using the given transport
// for authentication and TLS. The transport should already carry any
// required credentials (bearer token, client certs, exec-based auth, etc.).
func NewClient(baseURL string, rt http.RoundTripper) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: rt,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, form url.Values) ([]byte, error) {
	var reqBody io.Reader
	if method == http.MethodPost && form != nil {
		reqBody = strings.NewReader(form.Encode())
	}

	fullURL := c.baseURL + path
	if method == http.MethodGet && form != nil {
		fullURL += "?" + form.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	t0 := time.Now()
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(t0)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	klog.V(2).Infof("Prometheus %s %s completed in %.3fs (status %d)", method, path, elapsed.Seconds(), resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	return body, nil
}

// HTTPError represents an HTTP error response from Prometheus.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// InstantQuery executes an instant PromQL query (POST to avoid URL-length issues).
func (c *Client) InstantQuery(ctx context.Context, promql string) (map[string]interface{}, error) {
	form := url.Values{"query": {promql}}
	body, err := c.doRequest(ctx, http.MethodPost, "/api/v1/query", form)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

// RangeQuery executes a range PromQL query.
// start/end accept ISO-8601 timestamps, Unix epoch seconds, or relative offsets.
// Defaults to the last 1 hour when omitted.
func (c *Client) RangeQuery(ctx context.Context, promql, start, end, step string) (map[string]interface{}, error) {
	now := time.Now().UTC()
	if step == "" {
		step = "60s"
	}
	form := url.Values{
		"query": {promql},
		"start": {ResolveTime(start, now.Add(-1*time.Hour))},
		"end":   {ResolveTime(end, now)},
		"step":  {step},
	}
	body, err := c.doRequest(ctx, http.MethodPost, "/api/v1/query_range", form)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

// LabelValues returns all values for a label (metric names when label="__name__").
func (c *Client) LabelValues(ctx context.Context, label string) ([]string, error) {
	if label == "" {
		label = "__name__"
	}
	body, err := c.doRequest(ctx, http.MethodGet, "/api/v1/label/"+label+"/values", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return resp.Data, nil
}

// Labels returns all known label names.
func (c *Client) Labels(ctx context.Context) ([]string, error) {
	body, err := c.doRequest(ctx, http.MethodGet, "/api/v1/labels", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return resp.Data, nil
}

// Series returns label-sets for series matching the given selector.
func (c *Client) Series(ctx context.Context, match string) ([]map[string]string, error) {
	form := url.Values{"match[]": {match}}
	body, err := c.doRequest(ctx, http.MethodPost, "/api/v1/series", form)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []map[string]string `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return resp.Data, nil
}
