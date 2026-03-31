package table

import (
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

// renderMatrixPivotTable renders matrix entries as a pivot table: one column per
// unique label combination, one row per timestamp.
func renderMatrixPivotTable(title string, entries []entry, labelKeys []string, opts Options) string {
	type seriesData struct {
		values map[float64]string // timestamp -> formatted value
	}

	seriesMap := map[string]*seriesData{}
	var seriesOrder []string
	tsSet := map[float64]struct{}{}

	for _, e := range entries {
		colName := buildColumnName(e, labelKeys)

		sd, exists := seriesMap[colName]
		if !exists {
			sd = &seriesData{values: map[float64]string{}}
			seriesMap[colName] = sd
			seriesOrder = append(seriesOrder, colName)
		}

		for _, v := range e.values {
			pair, _ := v.([]interface{})
			if len(pair) != 2 {
				continue
			}
			ts, _ := pair[0].(float64)
			tsSet[ts] = struct{}{}
			sd.values[ts] = valueForFormat(pair[1], opts.Format)
		}
	}

	sort.Strings(seriesOrder)

	timestamps := make([]float64, 0, len(tsSet))
	for ts := range tsSet {
		timestamps = append(timestamps, ts)
	}
	sort.Float64s(timestamps)

	t := newTableWriter(title)

	if !(opts.NoHeaders && opts.Format != "markdown") {
		header := table.Row{"TIMESTAMP"}
		for _, col := range seriesOrder {
			header = append(header, col)
		}
		t.AppendHeader(header)
	}

	for _, ts := range timestamps {
		row := table.Row{opts.TimestampForFormat(ts)}
		for _, col := range seriesOrder {
			if sd, ok := seriesMap[col]; ok {
				row = append(row, sd.values[ts])
			} else {
				row = append(row, "")
			}
		}
		t.AppendRow(row)
	}

	return renderTableOutput(t, opts.Format)
}

// buildColumnName creates a human-readable column name from an entry's label
// values. When there is a single label key, the bare value is used (e.g. "pod-a").
// With multiple keys, values are joined with "/" (e.g. "ns1/pod-a").
func buildColumnName(e entry, labelKeys []string) string {
	parts := make([]string, 0, len(labelKeys))
	for _, k := range labelKeys {
		val, _ := e.metric[k].(string)
		if val != "" {
			parts = append(parts, val)
		}
	}
	if len(parts) == 0 {
		return "(no labels)"
	}
	return strings.Join(parts, "/")
}
