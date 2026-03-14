package table

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

// RenderMatrix renders a range-vector (matrix) Prometheus result as a pretty table.
// By default it uses a pivot layout (one column per label combination, one row
// per timestamp). Pass opts.NoPivot to revert to the traditional row-per-sample format.
func RenderMatrix(results []interface{}, opts Options) string {
	if len(results) == 0 {
		return "(no results)"
	}

	parsed := parseEntries(results)
	if len(parsed) == 0 {
		return "(no results)"
	}

	labelKeys := collectLabelKeys(parsed, opts.GroupBy)

	if opts.GroupBy != "" {
		return renderGrouped(parsed, labelKeys, opts, true)
	}

	if opts.NoPivot {
		return renderMatrixTable("", parsed, labelKeys, opts)
	}
	pivotKeys := append([]string{"__name__"}, labelKeys...)
	return renderMatrixPivotTable("", parsed, pivotKeys, opts)
}

// renderGrouped partitions entries by the GroupBy label and renders each group
// as a separate sub-table.
func renderGrouped(entries []entry, labelKeys []string, opts Options, matrix bool) string {
	groups := map[string][]entry{}
	var groupOrder []string

	for _, e := range entries {
		key, _ := e.metric[opts.GroupBy].(string)
		if key == "" {
			key = "(ungrouped)"
		}
		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], e)
	}

	sort.Strings(groupOrder)

	pivotKeys := append([]string{"__name__"}, labelKeys...)

	var sections []string
	for _, g := range groupOrder {
		title := fmt.Sprintf("--- %s: %s ---", opts.GroupBy, g)
		var rendered string
		switch {
		case matrix && !opts.NoPivot:
			rendered = renderMatrixPivotTable(title, groups[g], pivotKeys, opts)
		case matrix:
			rendered = renderMatrixTable(title, groups[g], labelKeys, opts)
		default:
			rendered = renderVectorTable(title, groups[g], labelKeys, opts)
		}
		sections = append(sections, rendered)
	}
	return strings.Join(sections, "\n\n")
}

func renderMatrixTable(title string, entries []entry, labelKeys []string, opts Options) string {
	t := newTableWriter(title)

	if !(opts.NoHeaders && opts.Format != "markdown") {
		header := table.Row{"METRIC"}
		for _, k := range labelKeys {
			header = append(header, strings.ToUpper(k))
		}
		header = append(header, "TIMESTAMP", "VALUE")
		t.AppendHeader(header)
	}

	for i, e := range entries {
		name := metricName(e, opts)
		labelVals := make([]string, len(labelKeys))
		for j, k := range labelKeys {
			labelVals[j], _ = e.metric[k].(string)
		}

		for _, v := range e.values {
			pair, _ := v.([]interface{})
			if len(pair) != 2 {
				continue
			}
			row := table.Row{name}
			for _, lv := range labelVals {
				row = append(row, lv)
			}
			ts, _ := pair[0].(float64)
			row = append(row, opts.FormatTimestamp(ts))
			row = append(row, formatValue(pair[1]))
			t.AppendRow(row)
		}

		if i < len(entries)-1 {
			t.AppendSeparator()
		}
	}

	return renderTableOutput(t, opts.Format)
}
