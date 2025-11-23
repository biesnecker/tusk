package cmd

import (
	"fmt"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/mastodon"
	"biesnecker.com/tusk/internal/output"
	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Display the latest post",
	Long:  `Display the latest post in your history (the one that -R and delete --latest would operate on).`,
	RunE:  runLatest,
}

func runLatest(cmd *cobra.Command, args []string) error {
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to open config store: %w", err)
	}
	defer store.Close()

	domain, _ := store.Get("domain")
	accessToken, _ := store.Get("access_token")

	if accessToken == "" {
		return fmt.Errorf("not authenticated. Run 'tusk auth' first")
	}

	client := mastodon.NewClient(domain, accessToken)

	// Get the last post ID from history
	lastPostID, err := store.GetLastPostID()
	if err != nil {
		return fmt.Errorf("failed to get last post ID: %w", err)
	}
	if lastPostID == "" {
		return fmt.Errorf("no posts in history")
	}

	// Fetch the status
	status, err := client.GetStatus(lastPostID)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Display the status
	output.Success("Latest post:")
	output.Plain("ID: %s", status.ID)
	output.URL(status.URL)
	output.Plain("")
	output.Plain("Content:")
	content := stripHTML(status.Content)
	output.Plain("%s", content)

	return nil
}
