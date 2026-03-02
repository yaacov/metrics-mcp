package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-metrics/pkg/help"
)

var helpPromqlCmd = &cobra.Command{
	Use:   "help-promql",
	Short: "Print PromQL quick reference",
	Long:  "Print a PromQL quick reference guide covering selectors, functions, aggregation, operators, and common patterns.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(help.GenerateHelp("promql"))
	},
}

func init() {
	rootCmd.AddCommand(helpPromqlCmd)
}
