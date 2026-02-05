package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"prompter-cli/internal/domain"
)

// Theme holds configurable colors for UI output.
type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	BasePrompt lipgloss.Color
	Border     lipgloss.Color
}

// ThemeFromConfig builds a theme with safe fallbacks.
func ThemeFromConfig(cfg domain.Config) Theme {
	return Theme{
		Primary:    resolveColor(cfg.Primary, "2"),
		Secondary:  resolveColor(cfg.Secondary, "5"),
		Accent:     resolveColor(cfg.Accent, "13"),
		BasePrompt: resolveColor(cfg.BasePrompt, "7"),
		Border:     resolveColor(cfg.Border, "8"),
	}
}

func resolveColor(value, fallback string) lipgloss.Color {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = fallback
	}
	return lipgloss.Color(trimmed)
}
