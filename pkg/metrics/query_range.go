package metrics

import (
	"context"
	"fmt"

	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
)

// NamedQuery pairs a PromQL expression with a display name.
type NamedQuery struct {
	Name  string
	Query string
}

// BuildNamedQueries pairs query and name slices into []NamedQuery.
// Missing names are auto-generated as q1, q2, ...
func BuildNamedQueries(queries, names []string) []NamedQuery {
	out := make([]NamedQuery, len(queries))
	for i, q := range queries {
		name := fmt.Sprintf("q%d", i+1)
		if i < len(names) && names[i] != "" {
			name = names[i]
		}
		out[i] = NamedQuery{Name: name, Query: q}
	}
	return out
}

// QueryRangeMulti executes one or more named range PromQL queries,
// overrides __name__ with each query's Name, merges the results,
// and returns formatted output.
func QueryRangeMulti(ctx context.Context, client *prometheus.Client, queries []NamedQuery, start, end, step, format string, opts ptable.Options) (string, error) {
	if len(queries) == 0 {
		return "Missing required flag 'query'. Provide one or more PromQL expressions.", nil
	}
	if step == "" {
		step = "60s"
	}

	merged := make([]interface{}, 0)
	for _, nq := range queries {
		if nq.Query == "" {
			continue
		}
		data, err := client.RangeQuery(ctx, nq.Query, start, end, step)
		if err != nil {
			return "", fmt.Errorf("query %q (%s): %w", nq.Name, nq.Query, err)
		}
		data, err = FilterData(data, opts.Selector)
		if err != nil {
			return "", err
		}
		results := extractResults(data)
		for _, r := range results {
			if m, ok := r.(map[string]interface{}); ok {
				metric, _ := m["metric"].(map[string]interface{})
				if metric == nil {
					metric = map[string]interface{}{}
					m["metric"] = metric
				}
				metric["__name__"] = nq.Name
			}
		}
		merged = append(merged, results...)
	}

	synthetic := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"resultType": "matrix",
			"result":     merged,
		},
	}
	return FormatResult(synthetic, format, opts), nil
}

// extractResults pulls the data.result slice from a Prometheus API response.
func extractResults(data map[string]interface{}) []interface{} {
	dataField, _ := data["data"].(map[string]interface{})
	if dataField == nil {
		return nil
	}
	results, _ := dataField["result"].([]interface{})
	return results
}
