package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/image"
	"biesnecker.com/tusk/internal/mastodon"
	"biesnecker.com/tusk/internal/output"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	replyTo     string
	replyLast   bool
	replyTUI    bool
	useEditor   bool
	visibility  string
	contentWarn string
	language    string
	dryRun      bool
	imagePath   string
	altText     string
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
  tusk post -R "Reply to last post"`,
	RunE: runPost,
}

func init() {
	postCmd.Flags().StringVarP(&replyTo, "reply", "r", "", "Reply to a specific status ID")
	postCmd.Flags().BoolVarP(&replyLast, "reply-last", "R", false, "Reply to the last posted status")
	postCmd.Flags().BoolVar(&replyTUI, "reply-tui", false, "Interactive TUI to select post to reply to")
	postCmd.Flags().BoolVarP(&useEditor, "editor", "e", false, "Compose post in $EDITOR")
	postCmd.Flags().StringVarP(&visibility, "visibility", "v", "public", "Post visibility (public, unlisted, private, direct)")
	postCmd.Flags().StringVarP(&contentWarn, "cw", "w", "", "Content warning / spoiler text")
	postCmd.Flags().StringVarP(&language, "lang", "l", "", "ISO 639 language code (e.g., en, es, fr, de, ja)")
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

	// Determine reply-to post first (before getting status text)
	var inReplyToID string
	if replyTUI {
		selectedID, err := runReplyTUI(store, client)
		if err != nil {
			return err
		}
		if selectedID == "" {
			return fmt.Errorf("no post selected")
		}
		inReplyToID = selectedID
	} else if replyLast {
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

	// Get status text after selecting reply-to post
	statusText, err := getStatusText(args, useEditor)
	if err != nil {
		return err
	}

	if statusText == "" {
		return fmt.Errorf("status text cannot be empty")
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
		Language:    language,
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
		if language != "" {
			output.Plain("Language: %s", language)
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

// TUI for selecting a post to reply to

type replyStatusItem struct {
	id      string
	content string
	url     string
}

type replySelectModel struct {
	store    *config.Store
	client   *mastodon.Client
	statuses []replyStatusItem
	cursor   int
	syncing  bool
	err      error
	selected bool
}

type replySyncCompleteMsg struct {
	err error
}

func loadReplyStatuses(store *config.Store, client *mastodon.Client) ([]replyStatusItem, error) {
	statuses, err := client.GetAccountStatuses(50)
	if err != nil {
		return nil, err
	}

	items := make([]replyStatusItem, 0, len(statuses))
	for _, status := range statuses {
		content := stripHTML(status.Content)
		items = append(items, replyStatusItem{
			id:      status.ID,
			content: content,
			url:     status.URL,
		})
	}

	return items, nil
}

func initialReplyModel(store *config.Store, client *mastodon.Client) replySelectModel {
	statuses, err := loadReplyStatuses(store, client)
	return replySelectModel{
		store:    store,
		client:   client,
		statuses: statuses,
		cursor:   0,
		err:      err,
		selected: false,
	}
}

func (m replySelectModel) Init() tea.Cmd {
	return nil
}

func doReplySync(store *config.Store, client *mastodon.Client) tea.Cmd {
	return func() tea.Msg {
		_, err := loadReplyStatuses(store, client)
		return replySyncCompleteMsg{err: err}
	}
}

func (m replySelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case replySyncCompleteMsg:
		m.syncing = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Reload statuses
			statuses, err := loadReplyStatuses(m.store, m.client)
			if err != nil {
				m.err = err
			} else {
				m.statuses = statuses
				m.cursor = 0
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.syncing {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.statuses)-1 {
				m.cursor++
			}

		case "s":
			// Sync posts
			m.syncing = true
			return m, doReplySync(m.store, m.client)

		case "enter", " ":
			// Select current post
			m.selected = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m replySelectModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.\n", m.err)
	}

	if m.syncing {
		return "Syncing posts...\n"
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	b.WriteString(headerStyle.Render("Select Post to Reply To"))
	b.WriteString("\n\n")

	// Instructions
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	b.WriteString(helpStyle.Render("↑/k: up  ↓/j: down  enter/space: select  s: sync  q: quit"))
	b.WriteString("\n\n")

	if len(m.statuses) == 0 {
		b.WriteString("No posts found.\n")
		return b.String()
	}

	// Posts
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	normalStyle := lipgloss.NewStyle()

	for i, status := range m.statuses {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		contentPreview := truncate(status.content, 80)
		line := fmt.Sprintf("%s %s", cursor, contentPreview)

		if m.cursor == i {
			line = cursorStyle.Render(line)
		} else {
			line = normalStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func runReplyTUI(store *config.Store, client *mastodon.Client) (string, error) {
	p := tea.NewProgram(initialReplyModel(store, client))
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running TUI: %w", err)
	}

	m := finalModel.(replySelectModel)
	if m.err != nil {
		return "", m.err
	}

	if !m.selected || len(m.statuses) == 0 {
		return "", nil
	}

	return m.statuses[m.cursor].id, nil
}

