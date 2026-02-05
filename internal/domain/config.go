package domain

import (
	"os"
	"path/filepath"
)

// Config describes the resolved configuration.
type Config struct {
	PromptsLocation         string            `toml:"prompts_location"`
	HistoryLocation         string            `toml:"history_location"`
	HistoryClearCycle       string            `toml:"history_clear_cycle"`
	HistoryFileFormat       string            `toml:"history_file_format"`
	HistoryEnableTimeAgo    bool              `toml:"history_enable_time_ago"`
	DisableHistory          bool              `toml:"disable_history"`
	LocalPromptsLocation    string            `toml:"local_prompts_location"`
	IncludeAgents           string            `toml:"include_agents"`
	Editor                  string            `toml:"editor"`
	DirectoryStrategy       string            `toml:"directory_strategy"`
	Target                  string            `toml:"target"`
	InteractiveDefault      bool              `toml:"interactive_default"`
	IncludeBuiltinShorthand bool              `toml:"include_builtin_shorthand"`
	RemapShortFlags         map[string]string `toml:"remap_short_flags"`
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
		HistoryEnableTimeAgo:    true,
		DisableHistory:          false,
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
