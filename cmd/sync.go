package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/djgrove/strava-attackpoint/internal/attackpoint"
	"github.com/djgrove/strava-attackpoint/internal/config"
	"github.com/djgrove/strava-attackpoint/internal/strava"
	"github.com/djgrove/strava-attackpoint/internal/sync"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	syncSince     string
	syncActivity  string
	syncCredsFile string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Strava activities to AttackPoint",
	Long: `Sync your Strava activities to AttackPoint.org.

Examples:
  strava-ap sync --since 2026-01-01    Sync all activities since January 1, 2026
  strava-ap sync --activity 12345      Re-sync a specific Strava activity`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringVar(&syncSince, "since", "", "Sync activities after this date (YYYY-MM-DD)")
	syncCmd.Flags().StringVar(&syncActivity, "activity", "", "Re-sync a specific Strava activity ID")
	syncCmd.Flags().StringVar(&syncCredsFile, "creds-file", "", "File with AP username on line 1, password on line 2")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	if syncSince == "" && syncActivity == "" {
		return fmt.Errorf("specify --since or --activity")
	}

	// Load config.
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if cfg.StravaClientID == "" {
		return fmt.Errorf("Strava not configured — run 'strava-ap setup' first")
	}

	// Load sync state.
	state, err := config.LoadSyncState()
	if err != nil {
		return fmt.Errorf("loading sync state: %w", err)
	}

	// Create Strava client.
	stravaClient, err := strava.NewClient(cfg)
	if err != nil {
		return err
	}

	// Get AP credentials.
	var apUsername, apPassword string
	if syncCredsFile != "" {
		apUsername, apPassword, err = readCredsFile(syncCredsFile)
	} else {
		apUsername, apPassword, err = promptAPCredentials()
	}
	if err != nil {
		return err
	}

	// Create and login AP client.
	apClient, err := attackpoint.NewClient()
	if err != nil {
		return err
	}
	fmt.Print("Logging in to AttackPoint... ")
	if err := apClient.Login(apUsername, apPassword); err != nil {
		return err
	}
	fmt.Println("done")

	// Create sync engine.
	engine := sync.NewEngine(stravaClient, apClient, state)

	if syncActivity != "" {
		activityID, err := strconv.ParseInt(syncActivity, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid activity ID: %w", err)
		}
		result, err := engine.SyncActivity(activityID)
		if err != nil {
			return err
		}
		printSummary([]sync.Result{*result})
		return nil
	}

	since, err := time.Parse("2006-01-02", syncSince)
	if err != nil {
		return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
	}

	results, err := engine.SyncSince(since)
	if err != nil {
		return err
	}

	printSummary(results)
	return nil
}

func readCredsFile(path string) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("opening creds file: %w", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())
	if username == "" || password == "" {
		return "", "", fmt.Errorf("creds file must have username on line 1, password on line 2")
	}
	return username, password, nil
}

func promptAPCredentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("AttackPoint username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("reading username: %w", err)
	}
	username = strings.TrimSpace(username)

	fmt.Print("AttackPoint password: ")
	// Try secure terminal reading first; fall back to plain stdin.
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		// Not a terminal — read from stdin directly.
		pw, readErr := reader.ReadString('\n')
		if readErr != nil {
			return "", "", fmt.Errorf("reading password: %w", readErr)
		}
		fmt.Println()
		return username, strings.TrimSpace(pw), nil
	}
	fmt.Println() // newline after hidden input

	return username, string(passwordBytes), nil
}

func printSummary(results []sync.Result) {
	if len(results) == 0 {
		fmt.Println("\nNo activities to sync.")
		return
	}

	synced, skipped, failed := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case "synced":
			synced++
		case "skipped":
			skipped++
		case "failed":
			failed++
		}
	}

	fmt.Printf("\nSync complete: %d synced, %d skipped, %d failed\n", synced, skipped, failed)

	if failed > 0 {
		fmt.Println("\nFailed activities:")
		for _, r := range results {
			if r.Status == "failed" {
				fmt.Printf("  - %s (ID: %d): %v\n", r.ActivityName, r.ActivityID, r.Error)
			}
		}
	}
}
