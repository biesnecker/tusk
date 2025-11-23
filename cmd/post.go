package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/image"
	"biesnecker.com/tusk/internal/mastodon"
	"biesnecker.com/tusk/internal/output"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	replyTo      string
	replyLast    bool
	deleteID     string
	deleteLast   bool
	forceDelete  bool
	useEditor    bool
	visibility   string
	contentWarn  string
	dryRun       bool
	imagePath    string
	altText      string
)

var postCmd = &cobra.Command{
	Use:   "post [TEXT]",
	Short: "Post a status to Mastodon",
	Long: `Post a status to Mastodon. You can post text directly, use an editor, or pipe from stdin.

Examples:
  tusk post "Hello, Mastodon!"
  tusk post -e
  echo "Hello" | tusk post
  tusk post -r STATUS_ID "This is a reply"
  tusk post -R "Reply to last post"
  tusk post -d STATUS_ID
  tusk post -D`,
	RunE: runPost,
}

func init() {
	postCmd.Flags().StringVarP(&replyTo, "reply", "r", "", "Reply to a specific status ID")
	postCmd.Flags().BoolVarP(&replyLast, "reply-last", "R", false, "Reply to the last posted status")
	postCmd.Flags().StringVarP(&deleteID, "delete", "d", "", "Delete a status by ID")
	postCmd.Flags().BoolVarP(&deleteLast, "delete-last", "D", false, "Delete the last posted status")
	postCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Skip confirmation when deleting")
	postCmd.Flags().BoolVarP(&useEditor, "editor", "e", false, "Compose post in $EDITOR")
	postCmd.Flags().StringVarP(&visibility, "visibility", "v", "public", "Post visibility (public, unlisted, private, direct)")
	postCmd.Flags().StringVarP(&contentWarn, "cw", "w", "", "Content warning / spoiler text")
	postCmd.Flags().StringVarP(&imagePath, "image", "i", "", "Path to image file to attach")
	postCmd.Flags().StringVar(&altText, "alt", "", "Alt text for the image")
	postCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be posted without actually posting")
}

func runPost(cmd *cobra.Command, args []string) error {
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

	if deleteLast {
		lastPostID, err := store.GetLastPostID()
		if err != nil {
			return fmt.Errorf("failed to get last post ID: %w", err)
		}
		if lastPostID == "" {
			return fmt.Errorf("no posts in history")
		}
		return handleDelete(client, store, lastPostID, forceDelete)
	}

	if deleteID != "" {
		return handleDelete(client, store, deleteID, forceDelete)
	}

	statusText, err := getStatusText(args)
	if err != nil {
		return err
	}

	if statusText == "" {
		return fmt.Errorf("status text cannot be empty")
	}

	var inReplyToID string
	if replyLast {
		lastPostID, err := store.GetLastPostID()
		if err != nil {
			return fmt.Errorf("failed to get last post ID: %w", err)
		}
		if lastPostID == "" {
			return fmt.Errorf("no last post found. Post something first or use -r to reply to a specific status")
		}
		inReplyToID = lastPostID
	} else if replyTo != "" {
		inReplyToID = replyTo
	}

	// Verify the status exists if replying
	if inReplyToID != "" {
		_, err := client.GetStatus(inReplyToID)
		if err != nil {
			return fmt.Errorf("failed to get status to reply to: %w", err)
		}
	}

	// Handle image upload
	var mediaIDs []string
	if imagePath != "" {
		// Check for alt text
		if altText == "" {
			output.Prompt("Warning: No alt text provided for image. Continue without alt text? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				output.Info("Post cancelled. Please add --alt \"your alt text\" and try again.")
				return nil
			}
		}

		// Process the image (convert HEIC, strip EXIF)
		output.Info("Processing image...")
		processedImage, err := image.ProcessImage(imagePath)
		if err != nil {
			return fmt.Errorf("failed to process image: %w", err)
		}

		// Upload the image
		output.Info("Uploading image...")
		media, err := client.UploadMedia(
			processedImage.Data,
			processedImage.Filename,
			processedImage.MimeType,
			altText,
		)
		if err != nil {
			return fmt.Errorf("failed to upload image: %w", err)
		}

		mediaIDs = []string{media.ID}
		output.Info("Image uploaded successfully")
	}

	params := mastodon.StatusParams{
		Status:      statusText,
		InReplyToID: inReplyToID,
		Visibility:  visibility,
		SpoilerText: contentWarn,
		MediaIDs:    mediaIDs,
	}

	if dryRun {
		output.Info("Dry run mode - would post:")
		output.Plain("Status: %s", statusText)
		if inReplyToID != "" {
			output.Plain("In reply to: %s", inReplyToID)
		}
		output.Plain("Visibility: %s", visibility)
		if contentWarn != "" {
			output.Plain("Content warning: %s", contentWarn)
		}
		if imagePath != "" {
			output.Plain("Image: %s", imagePath)
			if altText != "" {
				output.Plain("Alt text: %s", altText)
			}
		}
		return nil
	}

	output.Info("Posting status...")
	status, err := client.PostStatus(params)
	if err != nil {
		return fmt.Errorf("failed to post status: %w", err)
	}

	if err := store.AddPostToHistory(status.ID); err != nil {
		output.Error("Failed to save post to history: %v", err)
	}

	output.Success("Status posted!")
	output.URL(status.URL)

	return nil
}

func getStatusText(args []string) (string, error) {
	if useEditor {
		return getTextFromEditor()
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return getTextFromStdin()
	}

	if len(args) == 0 {
		return "", fmt.Errorf("no status text provided. Use an argument, -e for editor, or pipe from stdin")
	}

	return strings.Join(args, " "), nil
}

func getTextFromEditor() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	tmpFile, err := os.CreateTemp("", "tusk-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFilePath)

	cmd := exec.Command(editor, tmpFilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run editor: %w", err)
	}

	content, err := os.ReadFile(tmpFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read temp file: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

func getTextFromStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var builder strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				builder.WriteString(line)
				break
			}
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		builder.WriteString(line)
	}

	return strings.TrimSpace(builder.String()), nil
}

func handleDelete(client *mastodon.Client, store *config.Store, statusID string, force bool) error {
	if !force {
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
