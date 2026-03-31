// Package table provides pretty-printed table rendering for Prometheus query results.
package table

import (
	"strconv"
	"time"
)

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

// RawTimestamp returns the Unix epoch timestamp as a plain numeric string.
func (o Options) RawTimestamp(ts float64) string {
	return strconv.FormatFloat(ts, 'f', -1, 64)
}

// TimestampForFormat returns a raw epoch for machine formats (TSV, CSV)
// and a human-readable date for display formats (table, markdown).
func (o Options) TimestampForFormat(ts float64) string {
	if isMachineFormat(o.Format) {
		return o.RawTimestamp(ts)
	}
	return o.FormatTimestamp(ts)
}
