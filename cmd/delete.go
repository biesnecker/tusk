package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/mastodon"
	"biesnecker.com/tusk/internal/output"
	"github.com/spf13/cobra"
)

var (
	deleteLatest bool
	deleteForce  bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete [ID]",
	Short: "Delete a status",
	Long:  `Delete a status by ID or delete your most recent post.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteLatest, "latest", "l", false, "Delete the most recent post")
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation")
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	var statusID string

	if deleteLatest {
		lastPostID, err := store.GetLastPostID()
		if err != nil {
			return fmt.Errorf("failed to get last post ID: %w", err)
		}
		if lastPostID == "" {
			return fmt.Errorf("no posts in history")
		}
		statusID = lastPostID
	} else if len(args) == 1 {
		statusID = args[0]
	} else {
		return fmt.Errorf("must provide status ID or use --latest flag")
	}

	if !deleteForce {
		output.Prompt("Are you sure you want to delete status %s? This cannot be undone. (y/N): ", statusID)

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			output.Info("Deletion cancelled.")
			return nil
		}
	}

	output.Info("Deleting status...")
	if err := client.DeleteStatus(statusID); err != nil {
		return fmt.Errorf("failed to delete status: %w", err)
	}

	// Remove from post history
	if err := store.RemovePostFromHistory(statusID); err != nil {
		output.Error("Failed to remove post from history: %v", err)
	}

	output.Success("Status deleted!")
	return nil
}
