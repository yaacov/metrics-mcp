package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-metrics/pkg/connection"
	"github.com/yaacov/kubectl-metrics/pkg/metrics"
	"github.com/yaacov/kubectl-metrics/pkg/presets"
	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
)

var presetCmd = &cobra.Command{
	Use:   "preset",
	Short: "Run a pre-configured named PromQL query",
	Long: `Run a pre-configured named PromQL query.

Presets provide quick access to common cluster monitoring and MTV/Forklift
migration queries without writing PromQL. Many presets support namespace filtering.

Every preset works as both an instant query (default) and a range query.
Pass --start to get a time-series trend (e.g. --start "-1h").

Examples:
  kubectl metrics preset --name cluster_cpu_utilization
  kubectl metrics preset --name cluster_pod_status
  kubectl metrics preset --name mtv_migration_status
  kubectl metrics preset --name mtv_migration_pod_rx --namespace mtv-test --output json
  kubectl metrics preset --name mtv_net_throughput
  kubectl metrics preset --name mtv_net_throughput --start "-2h" --step "30s"

Available presets:` + formatPresetList(),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		promURL, rt, err := connection.ResolveConnection(ctx)
		if err != nil {
			return fmt.Errorf("TLS configuration error: %w", err)
		}
		if promURL == "" {
			return fmt.Errorf("Prometheus URL is required: use --url flag or ensure cluster access for auto-discovery")
		}

		name, _ := cmd.Flags().GetString("name")
		namespace, _ := cmd.Flags().GetString("namespace")
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
		client := prometheus.NewClient(promURL, rt)
		result, err := metrics.Preset(ctx, client, name, namespace, start, end, step, format, opts)
		if err != nil {
			return fmt.Errorf("%s", metrics.FriendlyError("preset", err, promURL))
		}
		fmt.Println(result)
		return nil
	},
}

func formatPresetList() string {
	s := "\n"
	for _, p := range presets.ListPresets() {
		s += fmt.Sprintf("  %-40s %s\n", p.Name, p.Description)
	}
	return s
}

func init() {
	presetCmd.Flags().String("name", "", "Preset name (required)")
	presetCmd.Flags().String("namespace", "", "Namespace filter")
	presetCmd.Flags().String("start", "", "Start time: enables range query (e.g. -1h, -7d). ISO-8601, Unix epoch, or relative offset.")
	presetCmd.Flags().String("end", "", "End time: same formats as start (default: now)")
	presetCmd.Flags().String("step", "", "Step interval for range queries (default: 60s). E.g. 15s, 5m, 1h.")
	presetCmd.Flags().StringP("output", "o", "markdown", "Output format: table, markdown, json, raw, csv, tsv")
	presetCmd.Flags().Bool("local-time", false, "Display timestamps in local timezone instead of UTC")
	presetCmd.Flags().String("group-by", "", "Label name to split results into sub-tables (e.g. namespace, pod)")
	presetCmd.Flags().Bool("no-pivot", false, "Disable pivot table layout for range results (show one row per sample instead)")
	presetCmd.Flags().Bool("no-headers", false, "Suppress header row in table, CSV, and TSV output")
	presetCmd.Flags().StringP("selector", "l", "", `Label selector to filter results (e.g. "namespace=prod,pod=~nginx.*")`)
	_ = presetCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(presetCmd)
}
