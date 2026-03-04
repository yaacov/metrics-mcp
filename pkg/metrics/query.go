package metrics

import (
	"context"

	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
)

// Query executes an instant PromQL query and returns formatted output.
func Query(ctx context.Context, client *prometheus.Client, promql, format string, opts ptable.Options) (string, error) {
	if promql == "" {
		return "Missing required flag 'query'. Provide a PromQL expression, e.g. --query \"up\"", nil
	}
	data, err := client.InstantQuery(ctx, promql)
	if err != nil {
		return "", err
	}
	data, err = FilterData(data, opts.Selector)
	if err != nil {
		return "", err
	}
	return FormatResult(data, format, opts), nil
}
