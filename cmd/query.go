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

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Execute an instant PromQL query",
	Long: `Execute an instant PromQL query against Prometheus/Thanos.

Returns the current value of the expression at a single point in time.

Examples:
  kubectl metrics query --query "up"
  kubectl metrics query --query "sum(rate(http_requests_total[5m])) by (code)" --output json
  kubectl metrics query --query "node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes * 100"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		promURL, rt := connection.ResolveConnection(ctx)
		if promURL == "" {
			return fmt.Errorf("Prometheus URL is required: use --url flag or ensure cluster access for auto-discovery")
		}

		query, _ := cmd.Flags().GetString("query")
		format, _ := cmd.Flags().GetString("output")
		name, _ := cmd.Flags().GetString("name")
		localTime, _ := cmd.Flags().GetBool("local-time")
		groupBy, _ := cmd.Flags().GetString("group-by")
		noPivot, _ := cmd.Flags().GetBool("no-pivot")
		noHeaders, _ := cmd.Flags().GetBool("no-headers")
		selector, _ := cmd.Flags().GetString("selector")

		opts := ptable.Options{
			MetricName: name,
			LocalTime:  localTime,
			GroupBy:    groupBy,
			NoPivot:    noPivot,
			NoHeaders:  noHeaders,
			Selector:   selector,
		}
		client := prometheus.NewClient(promURL, rt)
		result, err := metrics.Query(ctx, client, query, format, opts)
		if err != nil {
			return fmt.Errorf("%s", metrics.FriendlyError("query", err, promURL))
		}
		fmt.Println(result)
		return nil
	},
}

func init() {
	queryCmd.Flags().String("query", "", "PromQL expression (required)")
	queryCmd.Flags().StringP("output", "o", "markdown", "Output format: table, markdown, json, raw, csv, tsv")
	queryCmd.Flags().String("name", "", "Metric name to display in the first table column (optional)")
	queryCmd.Flags().Bool("local-time", false, "Display timestamps in local timezone instead of UTC")
	queryCmd.Flags().String("group-by", "", "Label name to split results into sub-tables (e.g. namespace, pod)")
	queryCmd.Flags().Bool("no-pivot", false, "Disable pivot table layout for range results (show one row per sample instead)")
	queryCmd.Flags().Bool("no-headers", false, "Suppress header row in table, CSV, and TSV output")
	queryCmd.Flags().StringP("selector", "l", "", `Label selector to filter results (e.g. "namespace=prod,pod=~nginx.*")`)
	_ = queryCmd.MarkFlagRequired("query")
	rootCmd.AddCommand(queryCmd)
}
