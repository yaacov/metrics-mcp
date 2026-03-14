// Package metrics provides shared logic for querying Prometheus and formatting results.
package metrics

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
)

// FormatResult formats a Prometheus API response in the given format (table, json, raw).
func FormatResult(data map[string]interface{}, format string, opts ptable.Options) string {
	if format == "" {
		format = "markdown"
	}
	if format == "raw" {
		b, _ := json.MarshalIndent(data, "", "  ")
		return string(b)
	}

	status, _ := data["status"].(string)
	if status != "success" {
		b, _ := json.MarshalIndent(data, "", "  ")
		return string(b)
	}

	dataField, _ := data["data"].(map[string]interface{})
	if dataField == nil {
		if format == "json" {
			return "[]"
		}
		return "(no results)"
	}
	resultSlice, _ := dataField["result"].([]interface{})
	if len(resultSlice) == 0 {
		if format == "json" {
			return "[]"
		}
		return "(no results)"
	}

	if format == "json" {
		b, _ := json.MarshalIndent(resultSlice, "", "  ")
		return string(b)
	}

	switch format {
	case "markdown":
		opts.Format = "markdown"
	case "csv":
		opts.Format = "csv"
	case "tsv":
		opts.Format = "tsv"
	default:
		opts.Format = "table"
	}

	resultType, _ := dataField["resultType"].(string)
	if resultType == "matrix" {
		return ptable.RenderMatrix(resultSlice, opts)
	}
	return ptable.RenderVector(resultSlice, opts)
}

// FriendlyError produces a user-friendly error message for Prometheus errors.
func FriendlyError(command string, err error, metricsURL string) string {
	if httpErr, ok := err.(*prometheus.HTTPError); ok {
		switch httpErr.StatusCode {
		case 401:
			return "Authentication failed (HTTP 401). Ensure a valid bearer token is provided via --token flag or Authorization header."
		case 403:
			return "Authorization denied (HTTP 403). The token may lack permissions to query this Prometheus endpoint."
		case 400:
			body := httpErr.Body
			if len(body) > 300 {
				body = body[:300]
			}
			return fmt.Sprintf("Bad request (HTTP 400) — likely an invalid PromQL expression.\n%s", body)
		case 422:
			body := httpErr.Body
			if len(body) > 300 {
				body = body[:300]
			}
			return fmt.Sprintf("Unprocessable query (HTTP 422).\n%s", body)
		case 503:
			return "Prometheus returned 503 (Service Unavailable). The server may be starting up or overloaded."
		default:
			body := httpErr.Body
			if len(body) > 300 {
				body = body[:300]
			}
			return fmt.Sprintf("HTTP %d from Prometheus.\n%s", httpErr.StatusCode, body)
		}
	}

	errStr := err.Error()
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		return fmt.Sprintf("Connection failed: cannot reach Prometheus at %q.\nCheck that the --url flag is correct and the server is reachable.", metricsURL)
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return fmt.Sprintf("Request timed out querying Prometheus at %q.\nThe query may be too expensive or the server is overloaded. Try a simpler query or a shorter time range.", metricsURL)
	}

	return fmt.Sprintf("Error in %s: %s", command, errStr)
}

// FlagStr extracts a string value from a flags map.
func FlagStr(flags map[string]any, key string) string {
	v, ok := flags[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	if s != "" {
		return s
	}
	if v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// FlagStrSlice extracts a string slice from a flags map.
// The value may be a single string or a JSON array of strings ([]interface{}).
func FlagStrSlice(flags map[string]any, key string) []string {
	v, ok := flags[key]
	if !ok {
		return nil
	}
	if s, ok := v.(string); ok {
		return []string{s}
	}
	if ss, ok := v.([]string); ok {
		return ss
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		} else if item != nil {
			out = append(out, fmt.Sprintf("%v", item))
		}
	}
	return out
}

// FlagBool extracts a boolean value from a flags map.
func FlagBool(flags map[string]any, key string) bool {
	v, ok := flags[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return strings.EqualFold(b, "true") || b == "1"
	case float64:
		return b != 0
	default:
		return false
	}
}
