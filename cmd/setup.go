package cmd

import (
	"github.com/djgrove/strava-attackpoint/internal/strava"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Connect to Strava",
	Long:  `Authorize strava-ap to access your Strava activities. Opens your browser to complete the OAuth flow.`,
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	return strava.RunOAuthFlow()
}
