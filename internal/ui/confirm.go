package ui

import "github.com/charmbracelet/lipgloss"

// ClipboardConfirm renders the standard clipboard confirmation message.
func ClipboardConfirm(theme Theme) string {
	style := lipgloss.NewStyle().
		Margin(1, 1).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Muted)).
		Foreground(lipgloss.Color(theme.Text)).
		Bold(true)

	return style.Render("  Copied to clipboard  󱁖")
}

// ExitMessage renders a standard framed exit message.
func ExitMessage(theme Theme, message string, mutedText bool) string {
	textColor := theme.Text
	bold := true
	if mutedText {
		textColor = theme.Muted
		bold = false
	}
	style := lipgloss.NewStyle().
		Margin(1, 1).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Muted)).
		Foreground(lipgloss.Color(textColor)).
		Bold(bold)

	return style.Render(message)
}
