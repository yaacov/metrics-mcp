package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-metrics/pkg/connection"
	"github.com/yaacov/kubectl-metrics/pkg/metrics"
	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
)

var queryRangeCmd = &cobra.Command{
	Use:   "query-range",
	Short: "Execute one or more range PromQL queries over a time window",
	Long: `Execute one or more range PromQL queries against Prometheus/Thanos.

Returns values over a time window (default: last 1 hour, 60s steps).
Use --query multiple times to run several queries in a single call.
Each query is labeled with the corresponding --name (or auto-generated q1, q2, ...).

Examples:
  kubectl metrics query-range --query "rate(http_requests_total[5m])" --start "-1h"
  kubectl metrics query-range --query "rate(container_cpu_usage_seconds_total[5m])" --query "container_memory_working_set_bytes" --name "cpu" --name "mem" --start "-1h"
  kubectl metrics query-range --query "node_cpu_seconds_total" --start "-7d" --step "1h" --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		promURL, rt, err := connection.ResolveConnection(ctx)
		if err != nil {
			return fmt.Errorf("TLS configuration error: %w", err)
		}
		if promURL == "" {
			return fmt.Errorf("Prometheus URL is required: use --url flag or ensure cluster access for auto-discovery")
		}

		querySlice, _ := cmd.Flags().GetStringArray("query")
		nameSlice, _ := cmd.Flags().GetStringArray("name")
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		step, _ := cmd.Flags().GetString("step")
		format, _ := cmd.Flags().GetString("output")
		localTime, _ := cmd.Flags().GetBool("local-time")
		groupBy, _ := cmd.Flags().GetString("group-by")
		noPivot, _ := cmd.Flags().GetBool("no-pivot")
		noHeaders, _ := cmd.Flags().GetBool("no-headers")
		selector, _ := cmd.Flags().GetString("selector")

		opts := ptable.Options{
			LocalTime: localTime,
			GroupBy:   groupBy,
			NoPivot:   noPivot,
			NoHeaders: noHeaders,
			Selector:  selector,
		}

		queries := metrics.BuildNamedQueries(querySlice, nameSlice)
		client := prometheus.NewClient(promURL, rt)
		result, err := metrics.QueryRangeMulti(ctx, client, queries, start, end, step, format, opts)
		if err != nil {
			return fmt.Errorf("%s", metrics.FriendlyError("query-range", err, promURL))
		}
		fmt.Println(result)
		return nil
	},
}

func init() {
	queryRangeCmd.Flags().StringArray("query", nil, "PromQL expression (required, repeatable)")
	queryRangeCmd.Flags().StringArray("name", nil, "Display name for each query (optional, repeatable)")
	queryRangeCmd.Flags().String("start", "", "Start time: ISO-8601, Unix epoch, or relative offset (default: -1h)")
	queryRangeCmd.Flags().String("end", "", "End time: same formats as start (default: now)")
	queryRangeCmd.Flags().String("step", "60s", "Step interval (e.g. 15s, 5m, 1h)")
	queryRangeCmd.Flags().StringP("output", "o", "markdown", "Output format: table, markdown, json, raw, csv, tsv")
	queryRangeCmd.Flags().Bool("local-time", false, "Display timestamps in local timezone instead of UTC")
	queryRangeCmd.Flags().String("group-by", "", "Label name to split results into sub-tables (e.g. namespace, pod)")
	queryRangeCmd.Flags().Bool("no-pivot", false, "Disable pivot table layout (show one row per sample instead)")
	queryRangeCmd.Flags().Bool("no-headers", false, "Suppress header row in table, CSV, and TSV output")
	queryRangeCmd.Flags().StringP("selector", "l", "", `Label selector to filter results (e.g. "namespace=prod,pod=~nginx.*")`)
	_ = queryRangeCmd.MarkFlagRequired("query")
	rootCmd.AddCommand(queryRangeCmd)
}
