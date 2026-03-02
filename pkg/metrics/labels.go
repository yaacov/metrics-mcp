package metrics

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
)

// Labels lists label names. If metric is specified, returns only labels
// that appear on series for that metric.
func Labels(ctx context.Context, client *prometheus.Client, metric string) (string, error) {
	if metric != "" {
		series, err := client.Series(ctx, metric)
		if err != nil {
			return "", err
		}
		if len(series) == 0 {
			return fmt.Sprintf("(no series found for %s)", metric), nil
		}
		labelSet := make(map[string]struct{})
		for _, s := range series {
			for k := range s {
				if k != "__name__" {
					labelSet[k] = struct{}{}
				}
			}
		}
		labels := make([]string, 0, len(labelSet))
		for k := range labelSet {
			labels = append(labels, k)
		}
		sort.Strings(labels)
		return strings.Join(labels, "\n"), nil
	}

	labels, err := client.Labels(ctx)
	if err != nil {
		return "", err
	}
	if len(labels) == 0 {
		return "(no labels found)", nil
	}
	return strings.Join(labels, "\n"), nil
}
