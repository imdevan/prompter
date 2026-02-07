package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// FrameStyle defines a shared container style for bordered views.
func FrameStyle(theme Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(1, 2).
		Margin(2, 1, 0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border)
}

// FrameSizeOptions configures shared list sizing inside framed views.
type FrameSizeOptions struct {
	HorizontalInset int
	VerticalInset   int
	MinWidth        int
	MinHeight       int
}

// ApplyFrameListSize clamps a list to fit a framed container.
func ApplyFrameListSize(model *list.Model, width, height int, opts FrameSizeOptions) {
	if model == nil {
		return
	}
	listWidth := width - opts.HorizontalInset
	listHeight := height - opts.VerticalInset
	if listWidth < opts.MinWidth {
		listWidth = opts.MinWidth
	}
	if listHeight < opts.MinHeight {
		listHeight = opts.MinHeight
	}
	model.SetSize(listWidth, listHeight)
}
