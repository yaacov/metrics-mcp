package table

import "sort"

type entry struct {
	metric map[string]interface{}
	value  []interface{} // [timestamp, value] for vector
	values []interface{} // [[timestamp, value], ...] for matrix
}

func parseEntries(results []interface{}) []entry {
	out := make([]entry, 0, len(results))
	for _, r := range results {
		m, _ := r.(map[string]interface{})
		if m == nil {
			continue
		}
		e := entry{
			metric: asStringMap(m["metric"]),
		}
		if v, ok := m["value"].([]interface{}); ok {
			e.value = v
		}
		if vs, ok := m["values"].([]interface{}); ok {
			e.values = vs
		}
		out = append(out, e)
	}
	return out
}

func asStringMap(v interface{}) map[string]interface{} {
	m, _ := v.(map[string]interface{})
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}

// collectLabelKeys returns sorted, deduplicated label keys across all entries,
// excluding __name__ and the groupBy label.
func collectLabelKeys(entries []entry, groupBy string) []string {
	seen := map[string]struct{}{}
	for _, e := range entries {
		for k := range e.metric {
			if k == "__name__" || (groupBy != "" && k == groupBy) {
				continue
			}
			seen[k] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func metricName(e entry, opts Options) string {
	if name, _ := e.metric["__name__"].(string); name != "" {
		return name
	}
	return opts.MetricName
}
