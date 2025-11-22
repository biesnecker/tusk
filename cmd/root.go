package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tusk",
	Short: "A CLI client for Mastodon",
	Long:  `Tusk is a command-line interface for interacting with Mastodon instances.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(postCmd)
	rootCmd.AddCommand(clearCmd)
	rootCmd.AddCommand(logoutCmd)
}
