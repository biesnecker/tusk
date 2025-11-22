package cmd

import (
	"fmt"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/mastodon"
	"biesnecker.com/tusk/internal/output"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and revoke access token",
	Long:  `Log out from the current Mastodon instance and revoke the access token.`,
	RunE:  runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to open config store: %w", err)
	}
	defer store.Close()

	accessToken, _ := store.Get("access_token")
	if accessToken == "" {
		output.Info("Not currently authenticated.")
		return nil
	}

	domain, _ := store.Get("domain")
	clientID, _ := store.Get("client_id")
	clientSecret, _ := store.Get("client_secret")

	client := mastodon.NewClient(domain, accessToken)

	output.Info("Revoking access token...")
	if err := client.RevokeToken(clientID, clientSecret); err != nil {
		output.Error("Failed to revoke token on server: %v", err)
		output.Info("Continuing to clear local data...")
	}

	if err := store.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear local data: %w", err)
	}

	output.Success("Logged out successfully!")
	return nil
}
