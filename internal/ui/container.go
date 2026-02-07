package ui

import "github.com/charmbracelet/lipgloss"

// FrameStyle defines a shared container style for bordered views.
func FrameStyle(theme Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(1, 2).
		Margin(1, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border)
}
