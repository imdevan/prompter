package testutil

import (
	"testing"

	"prompter-cli/internal/domain"
)

// ConfigOverrides lets tests override specific config fields.
type ConfigOverrides struct {
	PromptsLocation      *string
	HistoryLocation      *string
	LocalPromptsLocation *string
	Editor               *string
	Target               *string
	DisableHistory       *bool
}

// NewConfig returns a default config with optional overrides and temp XDG paths.
func NewConfig(t *testing.T, overrides ConfigOverrides) domain.Config {
	t.Helper()
	_, _, _ = WithTempXDG(t)
	cfg := domain.DefaultConfig()
	applyOverrides(&cfg, overrides)
	return cfg
}

func applyOverrides(cfg *domain.Config, overrides ConfigOverrides) {
	if overrides.PromptsLocation != nil {
		cfg.PromptsLocation = *overrides.PromptsLocation
	}
	if overrides.HistoryLocation != nil {
		cfg.HistoryLocation = *overrides.HistoryLocation
	}
	if overrides.LocalPromptsLocation != nil {
		cfg.LocalPromptsLocation = *overrides.LocalPromptsLocation
	}
	if overrides.Editor != nil {
		cfg.Editor = *overrides.Editor
	}
	if overrides.Target != nil {
		cfg.Target = *overrides.Target
	}
	if overrides.DisableHistory != nil {
		cfg.DisableHistory = *overrides.DisableHistory
	}
}
