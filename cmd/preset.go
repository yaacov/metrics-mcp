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

Many presets support a namespace filter. Use this command for quick access
to common MTV/Forklift monitoring queries without writing PromQL.

Presets marked [range] execute a range query over a time window with sensible
defaults. You can override the window with --start, --end, and --step flags.
Instant presets can also be promoted to range queries by passing --start.

Examples:
  kubectl metrics preset --name mtv_migration_status
  kubectl metrics preset --name mtv_migration_pod_rx --namespace mtv-test --format json
  kubectl metrics preset --name mtv_net_throughput_over_time
  kubectl metrics preset --name mtv_net_throughput_over_time --start "-2h" --step "30s"

Available presets:` + formatPresetList(),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		promURL, rt := connection.ResolveConnection(ctx)
		if promURL == "" {
			return fmt.Errorf("Prometheus URL is required: use --url flag or ensure cluster access for auto-discovery")
		}

		name, _ := cmd.Flags().GetString("name")
		namespace, _ := cmd.Flags().GetString("namespace")
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		step, _ := cmd.Flags().GetString("step")
		format, _ := cmd.Flags().GetString("format")
		localTime, _ := cmd.Flags().GetBool("local-time")
		groupBy, _ := cmd.Flags().GetString("group-by")

		opts := ptable.Options{
			LocalTime: localTime,
			GroupBy:   groupBy,
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
		s += fmt.Sprintf("  %-40s %-9s %s\n", p.Name, p.DisplayType(), p.Description)
	}
	return s
}

func init() {
	presetCmd.Flags().String("name", "", "Preset name (required)")
	presetCmd.Flags().String("namespace", "", "Namespace filter")
	presetCmd.Flags().String("start", "", "Start time: ISO-8601, Unix epoch, or relative offset (e.g. -1h). Overrides preset default for range presets; promotes instant presets to range.")
	presetCmd.Flags().String("end", "", "End time: same formats as start (default: now)")
	presetCmd.Flags().String("step", "", "Step interval (e.g. 15s, 5m, 1h). Overrides preset default.")
	presetCmd.Flags().String("format", "table", "Output format: table, markdown, json, raw")
	presetCmd.Flags().Bool("local-time", false, "Display timestamps in local timezone instead of UTC")
	presetCmd.Flags().String("group-by", "", "Label name to split results into sub-tables (e.g. namespace, pod)")
	_ = presetCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(presetCmd)
}
