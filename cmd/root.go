package cmd

import (
	"os"

	"github.com/djgrove/strava-attackpoint/internal/tui"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "strava-ap",
	Short: "Sync Strava activities to AttackPoint.org",
	Long:  "A tool that syncs your Strava activities to your AttackPoint.org training log.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
