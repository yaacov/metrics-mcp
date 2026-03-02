package table

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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

// RenderMatrix renders a range-vector (matrix) Prometheus result as a pretty table.
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
	return renderMatrixTable("", parsed, labelKeys, opts)
}

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

	var sections []string
	for _, g := range groupOrder {
		title := fmt.Sprintf("--- %s: %s ---", opts.GroupBy, g)
		var rendered string
		if matrix {
			rendered = renderMatrixTable(title, groups[g], labelKeys, opts)
		} else {
			rendered = renderVectorTable(title, groups[g], labelKeys, opts)
		}
		sections = append(sections, rendered)
	}
	return strings.Join(sections, "\n\n")
}

func formatValue(raw interface{}) string {
	s, _ := raw.(string)
	if s == "" {
		return fmt.Sprintf("%v", raw)
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return s
	}

	if f == 0 {
		return "0"
	}

	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}

	abs := math.Abs(f)
	if abs >= 1000 || abs < 0.001 {
		v, suffix := humanize.ComputeSI(f)
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".") + " " + suffix
	}

	return humanize.Ftoa(f)
}

func renderTable(t table.Writer, markdown bool) string {
	if markdown {
		return t.RenderMarkdown()
	}
	return t.Render()
}

func newTableWriter(title string) table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleDefault)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateHeader = true
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateFooter = false
	t.Style().Format.HeaderAlign = text.AlignLeft
	t.Style().Box.PaddingLeft = ""
	t.Style().Box.PaddingRight = "  "
	if title != "" {
		t.SetTitle(title)
	}
	return t
}

func renderVectorTable(title string, entries []entry, labelKeys []string, opts Options) string {
	t := newTableWriter(title)

	header := table.Row{"METRIC"}
	for _, k := range labelKeys {
		header = append(header, strings.ToUpper(k))
	}
	header = append(header, "TIMESTAMP", "VALUE")
	t.AppendHeader(header)

	for _, e := range entries {
		row := table.Row{metricName(e, opts)}
		for _, k := range labelKeys {
			val, _ := e.metric[k].(string)
			row = append(row, val)
		}
		if len(e.value) == 2 {
			ts, _ := e.value[0].(float64)
			row = append(row, opts.FormatTimestamp(ts))
			row = append(row, formatValue(e.value[1]))
		} else {
			row = append(row, "", "")
		}
		t.AppendRow(row)
	}

	return renderTable(t, opts.Markdown)
}

func renderMatrixTable(title string, entries []entry, labelKeys []string, opts Options) string {
	t := newTableWriter(title)

	header := table.Row{"METRIC"}
	for _, k := range labelKeys {
		header = append(header, strings.ToUpper(k))
	}
	header = append(header, "TIMESTAMP", "VALUE")
	t.AppendHeader(header)

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

	return renderTable(t, opts.Markdown)
}
