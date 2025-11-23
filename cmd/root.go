package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tusk [TEXT]",
	Short: "A CLI client for Mastodon",
	Long:  `Tusk is a command-line interface for interacting with Mastodon instances.`,
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided, run the post command
		return runPost(cmd, args)
	},
	// Disable flag parsing errors for unknown commands that might be text
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add all subcommands
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(postCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(clearCmd)
	rootCmd.AddCommand(logoutCmd)

	// Add post command flags to root command so they work without "post"
	rootCmd.Flags().StringVarP(&replyTo, "reply", "r", "", "Reply to a specific status ID")
	rootCmd.Flags().BoolVarP(&replyLast, "reply-last", "R", false, "Reply to the last posted status")
	rootCmd.Flags().BoolVarP(&useEditor, "editor", "e", false, "Compose post in $EDITOR")
	rootCmd.Flags().StringVarP(&visibility, "visibility", "v", "public", "Post visibility (public, unlisted, private, direct)")
	rootCmd.Flags().StringVarP(&contentWarn, "cw", "w", "", "Content warning / spoiler text")
	rootCmd.Flags().StringVarP(&imagePath, "image", "i", "", "Path to image file to attach")
	rootCmd.Flags().StringVar(&altText, "alt", "", "Alt text for the image")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be posted without actually posting")
}
