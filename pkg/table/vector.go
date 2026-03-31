package table

import (
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

// RenderVector renders an instant-vector Prometheus result as a pretty table.
func RenderVector(results []interface{}, opts Options) string {
	if len(results) == 0 {
		return "(no results)"
	}

	parsed := parseEntries(results)
	if len(parsed) == 0 {
		return "(no results)"
	}

	labelKeys := collectLabelKeys(parsed, opts.GroupBy)

	if opts.GroupBy != "" {
		return renderGrouped(parsed, labelKeys, opts, false)
	}
	return renderVectorTable("", parsed, labelKeys, opts)
}

func renderVectorTable(title string, entries []entry, labelKeys []string, opts Options) string {
	t := newTableWriter(title)

	if !(opts.NoHeaders && opts.Format != "markdown") {
		header := table.Row{"METRIC"}
		for _, k := range labelKeys {
			header = append(header, strings.ToUpper(k))
		}
		header = append(header, "TIMESTAMP", "VALUE")
		t.AppendHeader(header)
	}

	for _, e := range entries {
		row := table.Row{metricName(e, opts)}
		for _, k := range labelKeys {
			val, _ := e.metric[k].(string)
			row = append(row, val)
		}
		if len(e.value) == 2 {
			ts, _ := e.value[0].(float64)
			row = append(row, opts.TimestampForFormat(ts))
			row = append(row, valueForFormat(e.value[1], opts.Format))
		} else {
			row = append(row, "", "")
		}
		t.AppendRow(row)
	}

	return renderTableOutput(t, opts.Format)
}
