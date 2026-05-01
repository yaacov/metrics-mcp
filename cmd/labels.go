package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-metrics/pkg/connection"
	"github.com/yaacov/kubectl-metrics/pkg/metrics"
	"github.com/yaacov/kubectl-metrics/pkg/prometheus"
)

var labelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "List Prometheus labels",
	Long: `List Prometheus labels.

Without --metric: returns all known label names.
With --metric: returns the label keys that appear on series for that metric.

Examples:
  kubectl metrics labels
  kubectl metrics labels --metric container_network_receive_bytes_total`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		promURL, rt, err := connection.ResolveConnection(ctx)
		if err != nil {
			return fmt.Errorf("TLS configuration error: %w", err)
		}
		if promURL == "" {
			return fmt.Errorf("Prometheus URL is required: use --url flag or ensure cluster access for auto-discovery")
		}

		metric, _ := cmd.Flags().GetString("metric")

		client := prometheus.NewClient(promURL, rt)
		result, err := metrics.Labels(ctx, client, metric)
		if err != nil {
			return fmt.Errorf("%s", metrics.FriendlyError("labels", err, promURL))
		}
		fmt.Println(result)
		return nil
	},
}

func init() {
	labelsCmd.Flags().String("metric", "", "Metric name to inspect labels for")
	rootCmd.AddCommand(labelsCmd)
}
