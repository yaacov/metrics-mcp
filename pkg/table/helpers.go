package table

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

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

func rawValue(raw interface{}) string {
	s, _ := raw.(string)
	if s == "" {
		return fmt.Sprintf("%v", raw)
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return s
	}

	return strconv.FormatFloat(f, 'f', -1, 64)
}

func isMachineFormat(format string) bool {
	return format == "tsv" || format == "csv"
}

func valueForFormat(raw interface{}, format string) string {
	if isMachineFormat(format) {
		return rawValue(raw)
	}
	return formatValue(raw)
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

func renderTableOutput(t table.Writer, format string) string {
	switch strings.TrimSpace(strings.ToLower(format)) {
	case "markdown":
		return t.RenderMarkdown()
	case "csv":
		return t.RenderCSV()
	case "tsv":
		return t.RenderTSV()
	default:
		return t.Render()
	}
}
