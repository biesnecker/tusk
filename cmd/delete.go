package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"biesnecker.com/tusk/internal/config"
	"biesnecker.com/tusk/internal/mastodon"
	"biesnecker.com/tusk/internal/output"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	deleteLatest bool
	deleteForce  bool
	deleteTUI    bool
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
	deleteCmd.Flags().BoolVar(&deleteTUI, "tui", false, "Interactive TUI selection mode")
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

	// TUI mode
	if deleteTUI {
		return runDeleteTUI(store, client)
	}

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

// TUI model and methods

type statusItem struct {
	id       string
	content  string
	url      string
	selected bool
}

type deleteModel struct {
	store    *config.Store
	client   *mastodon.Client
	statuses []statusItem
	cursor   int
	syncing  bool
	err      error
	quitting bool
}

type syncCompleteMsg struct {
	err error
}

func loadStatuses(store *config.Store, client *mastodon.Client) ([]statusItem, error) {
	statuses, err := client.GetAccountStatuses(50)
	if err != nil {
		return nil, err
	}

	items := make([]statusItem, 0, len(statuses))
	for _, status := range statuses {
		content := stripHTML(status.Content)
		items = append(items, statusItem{
			id:       status.ID,
			content:  content,
			url:      status.URL,
			selected: false,
		})
	}

	return items, nil
}

func initialModel(store *config.Store, client *mastodon.Client) deleteModel {
	statuses, err := loadStatuses(store, client)
	return deleteModel{
		store:    store,
		client:   client,
		statuses: statuses,
		cursor:   0,
		err:      err,
	}
}

func (m deleteModel) Init() tea.Cmd {
	return nil
}

func doSync(store *config.Store, client *mastodon.Client) tea.Cmd {
	return func() tea.Msg {
		_, err := loadStatuses(store, client)
		return syncCompleteMsg{err: err}
	}
}

func (m deleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case syncCompleteMsg:
		m.syncing = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Reload statuses
			statuses, err := loadStatuses(m.store, m.client)
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
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.statuses)-1 {
				m.cursor++
			}

		case " ":
			if len(m.statuses) > 0 {
				m.statuses[m.cursor].selected = !m.statuses[m.cursor].selected
			}

		case "s":
			// Sync posts
			m.syncing = true
			return m, doSync(m.store, m.client)

		case "d":
			// Delete selected posts
			selectedIDs := []string{}
			for _, status := range m.statuses {
				if status.selected {
					selectedIDs = append(selectedIDs, status.id)
				}
			}

			if len(selectedIDs) == 0 {
				return m, nil
			}

			// Confirm deletion
			m.quitting = true
			return m, tea.Sequence(
				tea.Quit,
				func() tea.Msg {
					return deleteConfirmMsg{ids: selectedIDs}
				},
			)
		}
	}

	return m, nil
}

type deleteConfirmMsg struct {
	ids []string
}

func (m deleteModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.\n", m.err)
	}

	if m.syncing {
		return "Syncing posts...\n"
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	b.WriteString(headerStyle.Render("Delete Posts"))
	b.WriteString("\n\n")

	// Instructions
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	b.WriteString(helpStyle.Render("↑/k: up  ↓/j: down  space: toggle  s: sync  d: delete  q: quit"))
	b.WriteString("\n\n")

	if len(m.statuses) == 0 {
		b.WriteString("No posts found.\n")
		return b.String()
	}

	// Posts
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	normalStyle := lipgloss.NewStyle()

	for i, status := range m.statuses {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checkbox := "[ ]"
		if status.selected {
			checkbox = "[×]"
		}

		contentPreview := truncate(status.content, 80)

		line := fmt.Sprintf("%s %s %s", cursor, checkbox, contentPreview)

		if m.cursor == i {
			line = cursorStyle.Render(line)
		} else if status.selected {
			line = selectedStyle.Render(line)
		} else {
			line = normalStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func runDeleteTUI(store *config.Store, client *mastodon.Client) error {
	p := tea.NewProgram(initialModel(store, client))
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	m := finalModel.(deleteModel)
	if m.err != nil {
		return m.err
	}

	// Get selected IDs
	selectedIDs := []string{}
	for _, status := range m.statuses {
		if status.selected {
			selectedIDs = append(selectedIDs, status.id)
		}
	}

	if len(selectedIDs) == 0 {
		output.Info("No posts selected for deletion.")
		return nil
	}

	// Final confirmation
	output.Prompt("Delete %d post(s)? This cannot be undone. (y/N): ", len(selectedIDs))

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		output.Info("Deletion cancelled.")
		return nil
	}

	// Delete posts
	deletedCount := 0
	for _, id := range selectedIDs {
		output.Info("Deleting status %s...", id)
		if err := client.DeleteStatus(id); err != nil {
			output.Error("Failed to delete status %s: %v", id, err)
			continue
		}

		// Remove from post history
		if err := store.RemovePostFromHistory(id); err != nil {
			output.Error("Failed to remove post %s from history: %v", id, err)
		}

		deletedCount++
	}

	output.Success("Deleted %d post(s)!", deletedCount)
	return nil
}
