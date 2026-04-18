package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/djgrove/strava-attackpoint/internal/strava"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure Strava API credentials",
	Long: `Interactive setup wizard for connecting to Strava.

You'll need a Strava API application. Create one at:
  https://www.strava.com/settings/api

Set the "Authorization Callback Domain" to: localhost`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Strava API Setup ===")
	fmt.Println()
	fmt.Println("You need a Strava API application.")
	fmt.Println("Create one at: https://www.strava.com/settings/api")
	fmt.Println("Set 'Authorization Callback Domain' to: localhost")
	fmt.Println()

	fmt.Print("Client ID: ")
	clientID, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading client ID: %w", err)
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}

	fmt.Print("Client Secret: ")
	clientSecret, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading client secret: %w", err)
	}
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		return fmt.Errorf("client secret cannot be empty")
	}

	fmt.Println()
	return strava.RunOAuthFlow(clientID, clientSecret)
}
