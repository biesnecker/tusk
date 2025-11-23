package cmd

import (
	"fmt"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/mastodon"
	"biesnecker.com/tusk/internal/output"
	"github.com/spf13/cobra"
)

var syncLimit int

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync recent posts to local history",
	Long:  `Fetch your recent posts from Mastodon and add them to your local post history stack.`,
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().IntVarP(&syncLimit, "limit", "n", 50, "Number of recent posts to fetch (max 100)")
}

func runSync(cmd *cobra.Command, args []string) error {
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

	output.Info("Fetching your recent posts...")
	statuses, err := client.GetAccountStatuses(syncLimit)
	if err != nil {
		return fmt.Errorf("failed to fetch statuses: %w", err)
	}

	if len(statuses) == 0 {
		output.Info("No posts found to sync.")
		return nil
	}

	output.Info("Syncing %d posts to local history...", len(statuses))

	// Add statuses in reverse order (oldest first) so the newest is last in the stack
	syncedCount := 0
	for i := len(statuses) - 1; i >= 0; i-- {
		status := statuses[i]
		if err := store.AddPostToHistory(status.ID); err != nil {
			output.Error("Failed to add post %s to history: %v", status.ID, err)
		} else {
			syncedCount++
		}
	}

	output.Success("Synced %d posts to local history!", syncedCount)

	// Show the most recent post
	if len(statuses) > 0 {
		lastPost := statuses[0]
		output.Info("Most recent post: %s", lastPost.ID)
		if lastPost.URL != "" {
			output.Plain("  %s", lastPost.URL)
		}
	}

	return nil
}
