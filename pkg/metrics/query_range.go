package metrics

import (
	"context"

	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
)

// QueryRange executes a range PromQL query and returns formatted output.
func QueryRange(ctx context.Context, client *prometheus.Client, promql, start, end, step, format string, opts ptable.Options) (string, error) {
	if promql == "" {
		return "Missing required flag 'query'. Provide a PromQL expression, e.g. --query \"rate(http_requests_total[5m])\"", nil
	}
	if step == "" {
		step = "60s"
	}
	data, err := client.RangeQuery(ctx, promql, start, end, step)
	if err != nil {
		return "", err
	}
	return FormatResult(data, format, opts), nil
}
