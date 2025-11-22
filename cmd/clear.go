package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/output"
	"github.com/spf13/cobra"
)

var clearForce bool

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the post history stack",
	Long:  `Clear the local post history stack without logging out or affecting your authentication.`,
	RunE:  runClear,
}

func init() {
	clearCmd.Flags().BoolVarP(&clearForce, "force", "f", false, "Skip confirmation")
}

func runClear(cmd *cobra.Command, args []string) error {
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to open config store: %w", err)
	}
	defer store.Close()

	if !clearForce {
		output.Prompt("Are you sure you want to clear the post history? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			output.Info("Clear cancelled.")
			return nil
		}
	}

	if err := store.ClearPostHistory(); err != nil {
		return fmt.Errorf("failed to clear post history: %w", err)
	}

	output.Success("Post history cleared!")
	return nil
}
