package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/djgrove/strava-attackpoint/internal/attackpoint"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var debugCmd = &cobra.Command{
	Use:    "debug",
	Short:  "Debug AP form discovery",
	Hidden: true,
	RunE:   runDebug,
}

func init() {
	rootCmd.AddCommand(debugCmd)
}

func runDebug(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("AttackPoint username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("AttackPoint password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		pwStr, _ := reader.ReadString('\n')
		pw = []byte(strings.TrimSpace(pwStr))
	}
	fmt.Println()

	apClient, err := attackpoint.NewClient()
	if err != nil {
		return err
	}

	fmt.Print("Logging in... ")
	if err := apClient.Login(username, string(pw)); err != nil {
		return err
	}
	fmt.Println("done")

	fmt.Println("\nFetching /newtraining.jsp...")
	schema, err := apClient.DiscoverForm()
	if err != nil {
		return err
	}

	fmt.Printf("\nForm action: %q\n", schema.Action)
	fmt.Printf("Total fields discovered: %d\n\n", len(schema.Fields))

	for name, field := range schema.Fields {
		fmt.Printf("Field: %q (type: %s)\n", name, field.Type)
		if field.Type == "select" && len(field.Options) > 0 {
			for _, opt := range field.Options {
				fmt.Printf("  option: value=%q label=%q\n", opt.Value, opt.Label)
			}
		}
	}

	fmt.Printf("\nActivity types detected: %d\n", len(schema.ActivityTypes))
	for _, opt := range schema.ActivityTypes {
		fmt.Printf("  %q => %q\n", opt.Value, opt.Label)
	}

	// If user ID was discovered, fetch the log page and dump editutils/JS.
	if apClient.UserID != "" {
		fmt.Printf("\nUser ID: %s\n", apClient.UserID)
		fmt.Println("\nFetching log page to find edit/delete controls...")
		logResp, err := apClient.Get("/viewlog.jsp/user_" + apClient.UserID + "/period-7/enddate-2026-04-18")
		if err != nil {
			return fmt.Errorf("fetching log: %w", err)
		}
		defer logResp.Body.Close()
		logBody, _ := io.ReadAll(logResp.Body)
		logStr := string(logBody)

		// Look for editutils, delete, and relevant JS.
		for _, keyword := range []string{"editutils", "delete", "deletesession", "removeSession", "edittraining", "changetraining"} {
			idx := strings.Index(strings.ToLower(logStr), keyword)
			if idx >= 0 {
				start := idx - 100
				if start < 0 {
					start = 0
				}
				end := idx + 200
				if end > len(logStr) {
					end = len(logStr)
				}
				fmt.Printf("\n--- Found %q at pos %d ---\n%s\n", keyword, idx, logStr[start:end])
			}
		}

		// Also fetch the edit form for a specific session.
		fmt.Println("\nFetching /edittrainingsession.jsp?sessionid=9500442...")
		editResp, err := apClient.Get("/edittrainingsession.jsp?sessionid=9500442")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			defer editResp.Body.Close()
			editBody, _ := io.ReadAll(editResp.Body)
			editStr := string(editBody)
			// Look for delete links/buttons.
			for _, keyword := range []string{"delete", "remove", "form action"} {
				idx := strings.Index(strings.ToLower(editStr), keyword)
				if idx >= 0 {
					start := idx - 100
					if start < 0 {
						start = 0
					}
					end := idx + 300
					if end > len(editStr) {
						end = len(editStr)
					}
					fmt.Printf("\n--- Edit form: found %q ---\n%s\n", keyword, editStr[start:end])
				}
			}
		}
	}

	return nil
}
