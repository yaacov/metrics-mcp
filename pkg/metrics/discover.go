package metrics

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
)

// Discover lists available metric names, optionally filtered by keyword
// and grouped by prefix.
func Discover(ctx context.Context, client *prometheus.Client, keyword string, groupByPrefix bool) (string, error) {
	names, err := client.LabelValues(ctx, "__name__")
	if err != nil {
		return "", err
	}

	if keyword != "" {
		keyword = strings.ToLower(keyword)
		filtered := make([]string, 0)
		for _, n := range names {
			if strings.Contains(strings.ToLower(n), keyword) {
				filtered = append(filtered, n)
			}
		}
		names = filtered
	}

	if groupByPrefix {
		prefixCounts := make(map[string]int)
		for _, n := range names {
			parts := strings.SplitN(n, "_", 3)
			var prefix string
			if len(parts) >= 2 {
				prefix = parts[0] + "_" + parts[1]
			} else {
				prefix = parts[0]
			}
			prefixCounts[prefix]++
		}

		type kv struct {
			key   string
			count int
		}
		sorted := make([]kv, 0, len(prefixCounts))
		for k, v := range prefixCounts {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })

		var lines []string
		limit := 30
		if len(sorted) < limit {
			limit = len(sorted)
		}
		for _, kv := range sorted[:limit] {
			lines = append(lines, fmt.Sprintf("%-45s %4d metrics", kv.key, kv.count))
		}
		if len(lines) == 0 {
			return "(no metrics found)", nil
		}
		return strings.Join(lines, "\n"), nil
	}

	if len(names) == 0 {
		return "(no metrics found)", nil
	}
	return strings.Join(names, "\n"), nil
}
