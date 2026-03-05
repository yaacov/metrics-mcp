package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaacov/kubectl-metrics/pkg/presets"
	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
)

// Preset executes a pre-configured named query and returns formatted output.
// For range presets, start/end/step override the preset defaults when non-empty.
func Preset(ctx context.Context, client *prometheus.Client, name, namespace, start, end, step, format string, opts ptable.Options) (string, error) {
	if name == "" {
		return "Missing required flag 'name'. Use 'kubectl metrics preset --help' to list available presets.", nil
	}
	p, ok := presets.GetPreset(name, namespace)
	if !ok {
		available := make([]string, 0)
		for _, pr := range presets.ListPresets() {
			available = append(available, pr.Name)
		}
		return fmt.Sprintf("Unknown preset: %s\nAvailable: %s", name, strings.Join(available, ", ")), nil
	}

	useRange := start != ""
	if useRange && step == "" {
		step = "60s"
	}

	var data map[string]interface{}
	var err error
	if useRange {
		data, err = client.RangeQuery(ctx, p.Query, start, end, step)
	} else {
		data, err = client.InstantQuery(ctx, p.Query)
	}
	if err != nil {
		return "", err
	}
	data, err = FilterData(data, opts.Selector)
	if err != nil {
		return "", err
	}

	if format == "json" || format == "raw" {
		envelope := map[string]interface{}{
			"description": p.Description,
			"query":       p.Query,
		}
		if useRange {
			envelope["type"] = "range"
			envelope["start"] = start
			envelope["step"] = step
		}
		if format == "raw" {
			envelope["data"] = data
		} else {
			dataField, _ := data["data"].(map[string]interface{})
			if dataField != nil {
				if resultSlice, _ := dataField["result"].([]interface{}); resultSlice != nil {
					envelope["data"] = resultSlice
				} else {
					envelope["data"] = []interface{}{}
				}
			} else {
				envelope["data"] = []interface{}{}
			}
		}
		b, _ := json.MarshalIndent(envelope, "", "  ")
		return string(b), nil
	}

	opts.MetricName = name
	header := fmt.Sprintf("# %s\n# query: %s\n\n", p.Description, p.Query)
	return header + FormatResult(data, format, opts), nil
}
