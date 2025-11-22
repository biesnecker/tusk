package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/mastodon"
	"biesnecker.com/tusk/internal/oauth"
	"biesnecker.com/tusk/internal/output"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with a Mastodon instance",
	Long:  `Authenticate with a Mastodon instance to obtain an access token.`,
	RunE:  runAuth,
}

func runAuth(cmd *cobra.Command, args []string) error {
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to open config store: %w", err)
	}
	defer store.Close()

	existingToken, _ := store.Get("access_token")
	if existingToken != "" {
		output.Info("You are already authenticated.")
		output.Prompt("Do you want to re-authenticate? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			output.Info("Authentication cancelled.")
			return nil
		}
	}

	output.Prompt("Enter your Mastodon instance domain (e.g., mastodon.social): ")
	reader := bufio.NewReader(os.Stdin)
	domain, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read domain: %w", err)
	}
	domain = strings.TrimSpace(domain)

	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}

	output.Info("Starting OAuth flow...")

	callbackServer, err := oauth.NewCallbackServer()
	if err != nil {
		return fmt.Errorf("failed to create callback server: %w", err)
	}

	client := mastodon.NewClient(domain, "")
	redirectURI := callbackServer.RedirectURI()

	output.Info("Registering application...")
	app, err := client.RegisterApp("Tusk CLI", redirectURI, "read write follow")
	if err != nil {
		return fmt.Errorf("failed to register app: %w", err)
	}

	if err := store.Set("domain", domain); err != nil {
		return fmt.Errorf("failed to save domain: %w", err)
	}
	if err := store.Set("client_id", app.ClientID); err != nil {
		return fmt.Errorf("failed to save client_id: %w", err)
	}
	if err := store.Set("client_secret", app.ClientSecret); err != nil {
		return fmt.Errorf("failed to save client_secret: %w", err)
	}

	authURL := client.GetAuthorizationURL(app.ClientID, redirectURI, "read write follow")

	if err := callbackServer.Start(); err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}

	output.Info("Opening browser for authorization...")
	output.Plain("If the browser doesn't open automatically, visit this URL:")
	output.URL(authURL)

	if err := oauth.OpenBrowser(authURL); err != nil {
		output.Error("Failed to open browser automatically: %v", err)
	}

	output.Info("Waiting for authorization (timeout: 5 minutes)...")

	code, err := callbackServer.WaitForCode(5 * time.Minute)
	if err != nil {
		return fmt.Errorf("failed to get authorization code: %w", err)
	}

	output.Info("Exchanging code for access token...")
	accessToken, err := client.GetAccessToken(app.ClientID, app.ClientSecret, redirectURI, code)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	if err := store.Set("access_token", accessToken); err != nil {
		return fmt.Errorf("failed to save access token: %w", err)
	}

	output.Success("Authentication successful!")
	return nil
}
