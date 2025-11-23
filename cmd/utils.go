package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/mattn/go-isatty"
)

// stripHTML removes HTML tags and decodes common HTML entities
func stripHTML(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Collapse multiple spaces and newlines
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text
}

// truncate truncates a string to maxLen, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// getStatusText gets status text from args, editor, or stdin
func getStatusText(args []string, useEditor bool) (string, error) {
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

// getTextFromEditor opens the user's $EDITOR to compose text
func getTextFromEditor() (string, error) {
	return getTextFromEditorWithInitial("")
}

// getTextFromEditorWithInitial opens the user's $EDITOR with initial content
func getTextFromEditorWithInitial(initialContent string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	tmpFile, err := os.CreateTemp("", "tusk-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFilePath := tmpFile.Name()

	// Write initial content if provided
	if initialContent != "" {
		if err := os.WriteFile(tmpFilePath, []byte(initialContent), 0600); err != nil {
			tmpFile.Close()
			os.Remove(tmpFilePath)
			return "", fmt.Errorf("failed to write initial content: %w", err)
		}
	}

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

// getTextFromStdin reads text from stdin
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
