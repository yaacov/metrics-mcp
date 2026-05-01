package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-metrics/pkg/connection"
	"github.com/yaacov/kubectl-metrics/pkg/metrics"
	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "List available Prometheus metric names",
	Long: `List available Prometheus metric names.

Retrieves all metric names from Prometheus. Optionally filter by keyword or group by prefix.

Examples:
  kubectl metrics discover
  kubectl metrics discover --keyword mtv
  kubectl metrics discover --keyword network --group-by-prefix`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		promURL, rt, err := connection.ResolveConnection(ctx)
		if err != nil {
			return fmt.Errorf("TLS configuration error: %w", err)
		}
		if promURL == "" {
			return fmt.Errorf("Prometheus URL is required: use --url flag or ensure cluster access for auto-discovery")
		}

		keyword, _ := cmd.Flags().GetString("keyword")
		groupByPrefix, _ := cmd.Flags().GetBool("group-by-prefix")

		client := prometheus.NewClient(promURL, rt)
		result, err := metrics.Discover(ctx, client, keyword, groupByPrefix)
		if err != nil {
			return fmt.Errorf("%s", metrics.FriendlyError("discover", err, promURL))
		}
		fmt.Println(result)
		return nil
	},
}

func init() {
	discoverCmd.Flags().String("keyword", "", "Case-insensitive substring filter")
	discoverCmd.Flags().Bool("group-by-prefix", false, "Group metrics by two-part prefix and return counts")
	rootCmd.AddCommand(discoverCmd)
}
