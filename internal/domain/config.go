package domain

import (
	"os"
	"path/filepath"
)

// Config describes the resolved configuration.
type Config struct {
	PromptsLocation         string
	HistoryLocation         string
	HistoryClearCycle       string
	HistoryFileFormat       string
	LocalPromptsLocation    string
	IncludeAgents           string
	Editor                  string
	DirectoryStrategy       string
	Target                  string
	InteractiveDefault      bool
	IncludeBuiltinShorthand bool
	RemapShortFlags         map[string]string
}

// DefaultConfig returns the default configuration values.
func DefaultConfig() Config {
	dataHome := xdgHome("XDG_DATA_HOME", filepath.Join(".local", "share"))
	cacheHome := xdgHome("XDG_CACHE_HOME", ".cache")

	return Config{
		PromptsLocation:         filepath.Join(dataHome, "prompter", "prompts"),
		HistoryLocation:         filepath.Join(cacheHome, "prompter", "history"),
		HistoryClearCycle:       "never",
		HistoryFileFormat:       "month-day_eu",
		LocalPromptsLocation:    "prompts",
		IncludeAgents:           "all",
		Editor:                  "nvim",
		DirectoryStrategy:       "git",
		Target:                  "clipboard",
		InteractiveDefault:      true,
		IncludeBuiltinShorthand: true,
		RemapShortFlags:         map[string]string{},
	}
}

func xdgHome(envKey, fallbackSuffix string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, fallbackSuffix)
}
