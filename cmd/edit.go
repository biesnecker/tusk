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
	editLatest    bool
	editTUI       bool
	editEditor    bool
	editVisibility string
	editContentWarn string
	editLanguage    string
	editImagePath   string
	editAltText     string
)

var editCmd = &cobra.Command{
	Use:   "edit [ID] [TEXT]",
	Short: "Edit a status",
	Long:  `Edit a status by ID, your most recent post, or select interactively via TUI.`,
	Args:  cobra.MinimumNArgs(0),
	RunE:  runEdit,
}

func init() {
	editCmd.Flags().BoolVarP(&editLatest, "latest", "l", false, "Edit the most recent post")
	editCmd.Flags().BoolVar(&editTUI, "tui", false, "Interactive TUI selection mode")
	editCmd.Flags().BoolVarP(&editEditor, "editor", "e", false, "Compose edit in $EDITOR")
	editCmd.Flags().StringVarP(&editVisibility, "visibility", "v", "", "Post visibility (public, unlisted, private, direct)")
	editCmd.Flags().StringVarP(&editContentWarn, "cw", "w", "", "Content warning / spoiler text")
	editCmd.Flags().StringVar(&editLanguage, "lang", "", "ISO 639 language code (e.g., en, es, fr, de, ja)")
	editCmd.Flags().StringVarP(&editImagePath, "image", "i", "", "Path to image file to attach")
	editCmd.Flags().StringVar(&editAltText, "alt", "", "Alt text for the image")
}

func runEdit(cmd *cobra.Command, args []string) error {
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

	// Determine which status to edit
	var statusID string

	if editTUI {
		selectedID, err := runEditTUI(store, client)
		if err != nil {
			return err
		}
		if selectedID == "" {
			return fmt.Errorf("no post selected")
		}
		statusID = selectedID
	} else if editLatest {
		lastPostID, err := store.GetLastPostID()
		if err != nil {
			return fmt.Errorf("failed to get last post ID: %w", err)
		}
		if lastPostID == "" {
			return fmt.Errorf("no posts in history")
		}
		statusID = lastPostID
	} else if len(args) > 0 {
		statusID = args[0]
		args = args[1:] // Remove the ID from args for status text extraction
	} else {
		return fmt.Errorf("must provide status ID, use --latest, or use --tui")
	}

	// Get the current status to check it exists and get its content
	currentStatus, err := client.GetStatus(statusID)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Get current status text (stripped of HTML)
	currentText := stripHTML(currentStatus.Content)

	// Get status text
	var statusText string
	if editEditor {
		// Pre-populate editor with current content
		statusText, err = getTextFromEditorWithInitial(currentText)
		if err != nil {
			return err
		}
	} else if len(args) > 0 || !isTerminal() {
		statusText, err = getStatusText(args, false)
		if err != nil {
			return err
		}
	}

	// If no text provided, use current status text
	if statusText == "" {
		statusText = currentText
	}

	// Handle image upload
	var mediaIDs []string
	if editImagePath != "" {
		// User is providing a new image - upload it
		// Check for alt text
		if editAltText == "" {
			output.Prompt("Warning: No alt text provided for image. Continue without alt text? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				output.Info("Edit cancelled. Please add --alt \"your alt text\" and try again.")
				return nil
			}
		}

		// Process the image (convert HEIC, strip EXIF)
		output.Info("Processing image...")
		processedImage, err := image.ProcessImage(editImagePath)
		if err != nil {
			return fmt.Errorf("failed to process image: %w", err)
		}

		// Upload the image
		output.Info("Uploading image...")
		media, err := client.UploadMedia(
			processedImage.Data,
			processedImage.Filename,
			processedImage.MimeType,
			editAltText,
		)
		if err != nil {
			return fmt.Errorf("failed to upload image: %w", err)
		}

		mediaIDs = []string{media.ID}
		output.Info("Image uploaded successfully")
	} else {
		// No new image provided - preserve existing media attachments
		if len(currentStatus.MediaAttachments) > 0 {
			for _, attachment := range currentStatus.MediaAttachments {
				mediaIDs = append(mediaIDs, attachment.ID)
			}
		}
	}

	params := mastodon.StatusParams{
		Status:      statusText,
		Visibility:  editVisibility,
		SpoilerText: editContentWarn,
		MediaIDs:    mediaIDs,
		Language:    editLanguage,
	}

	output.Info("Editing status...")
	status, err := client.EditStatus(statusID, params)
	if err != nil {
		return fmt.Errorf("failed to edit status: %w", err)
	}

	output.Success("Status edited!")
	output.URL(status.URL)

	return nil
}

// Helper to check if stdin is a terminal
func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// TUI for selecting a post to edit

type editStatusItem struct {
	id       string
	content  string
	url      string
	selected bool
}

type editSelectModel struct {
	store    *config.Store
	client   *mastodon.Client
	statuses []editStatusItem
	cursor   int
	syncing  bool
	err      error
	selected bool
}

type editSyncCompleteMsg struct {
	err error
}

func loadEditStatuses(store *config.Store, client *mastodon.Client) ([]editStatusItem, error) {
	statuses, err := client.GetAccountStatuses(50)
	if err != nil {
		return nil, err
	}

	items := make([]editStatusItem, 0, len(statuses))
	for _, status := range statuses {
		content := stripHTML(status.Content)
		items = append(items, editStatusItem{
			id:       status.ID,
			content:  content,
			url:      status.URL,
			selected: false,
		})
	}

	return items, nil
}

func initialEditModel(store *config.Store, client *mastodon.Client) editSelectModel {
	statuses, err := loadEditStatuses(store, client)
	return editSelectModel{
		store:    store,
		client:   client,
		statuses: statuses,
		cursor:   0,
		err:      err,
		selected: false,
	}
}

func (m editSelectModel) Init() tea.Cmd {
	return nil
}

func doEditSync(store *config.Store, client *mastodon.Client) tea.Cmd {
	return func() tea.Msg {
		_, err := loadEditStatuses(store, client)
		return editSyncCompleteMsg{err: err}
	}
}

func (m editSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editSyncCompleteMsg:
		m.syncing = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Reload statuses
			statuses, err := loadEditStatuses(m.store, m.client)
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
			return m, doEditSync(m.store, m.client)

		case "enter", " ":
			// Select current post
			m.selected = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m editSelectModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.\n", m.err)
	}

	if m.syncing {
		return "Syncing posts...\n"
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	b.WriteString(headerStyle.Render("Select Post to Edit"))
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

func runEditTUI(store *config.Store, client *mastodon.Client) (string, error) {
	p := tea.NewProgram(initialEditModel(store, client))
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running TUI: %w", err)
	}

	m := finalModel.(editSelectModel)
	if m.err != nil {
		return "", m.err
	}

	if !m.selected || len(m.statuses) == 0 {
		return "", nil
	}

	return m.statuses[m.cursor].id, nil
}
