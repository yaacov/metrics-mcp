// Package table provides pretty-printed table rendering for Prometheus query results.
package table

import "time"

// Options controls how Prometheus results are rendered as tables.
type Options struct {
	// MetricName is used as the METRIC column value when __name__ is absent.
	MetricName string

	// LocalTime renders timestamps in the local timezone instead of UTC.
	LocalTime bool

	// GroupBy is a label name used to partition results into sub-tables
	// (e.g. "namespace", "pod"). Empty means no grouping.
	GroupBy string

	// Format selects the table output style: "table" (default), "markdown",
	// "csv", or "tsv".
	Format string

	// NoHeaders suppresses the header row in table, CSV, and TSV output.
	NoHeaders bool

	// NoPivot disables the default pivot layout for matrix results.
	// When false (default), range queries render one column per label
	// combination and one row per timestamp. When true, the traditional
	// row-per-sample format is used.
	NoPivot bool

	// Selector is a Kubernetes-style label selector that filters results
	// post-query (e.g. "namespace=prod,pod=~nginx.*"). Empty means no filtering.
	Selector string
}

const dateFormat = "2006-01-02 15:04:05"

// FormatTimestamp converts a Prometheus Unix timestamp to a human-readable string.
func (o Options) FormatTimestamp(ts float64) string {
	t := time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
	if !o.LocalTime {
		t = t.UTC()
	}
	return t.Format(dateFormat)
}
