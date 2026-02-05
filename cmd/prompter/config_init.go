package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"prompter-cli/internal/adapters/editor"
	"prompter-cli/internal/config"
	"prompter-cli/internal/domain"
	"prompter-cli/internal/utils"
)

type configInitOptions struct {
	force        bool
	openInEditor bool
}

func newConfigInitCmd() *cobra.Command {
	opts := &configInitOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a default config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigInit(cmd, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "overwrite existing config")
	cmd.Flags().BoolVarP(&opts.openInEditor, "editor", "e", false, "open config in editor after creation")
	return cmd
}

func runConfigInit(cmd *cobra.Command, opts *configInitOptions) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	manager := config.NewManager(cwd)
	exists, err := manager.Exists()
	if err != nil {
		return err
	}
	if exists && !opts.force {
		return fmt.Errorf("config already exists at %s (use --force to overwrite)", utils.ConfigPathGlobal())
	}
	cfg := domain.DefaultConfig()
	path := utils.ConfigPathGlobal()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := renderConfigTemplate(cfg)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if opts.openInEditor {
		editorAdapter := editor.New(cfg.Editor)
		if err := editorAdapter.Open(path); err != nil {
			return err
		}
	}
	cmd.Printf("Wrote config to %s\n", utils.ConfigPathGlobal())
	return nil
}

func renderConfigTemplate(cfg domain.Config) string {
	var builder strings.Builder
	builder.WriteString("# Locations\n")
	builder.WriteString(fmt.Sprintf("prompts_location = %q\n", cfg.PromptsLocation))
	builder.WriteString(fmt.Sprintf("history_location = %q\n", cfg.HistoryLocation))
	builder.WriteString(fmt.Sprintf("local_prompts_location = %q\n", cfg.LocalPromptsLocation))
	builder.WriteString("\n# Templates & agents\n")
	builder.WriteString("# include_agents options: all, none, agents, kiro, cursor\n")
	builder.WriteString(fmt.Sprintf("include_agents = %q\n", cfg.IncludeAgents))
	builder.WriteString(fmt.Sprintf("editor = %q\n", cfg.Editor))
	builder.WriteString("# directory_strategy options: git (tracked files), filesystem (walk directory, uses .gitignore when present)\n")
	builder.WriteString(fmt.Sprintf("directory_strategy = %q\n", cfg.DirectoryStrategy))
	builder.WriteString("\n# Output\n")
	builder.WriteString("# target options: clipboard, stdout, file:/path, editor\n")
	builder.WriteString(fmt.Sprintf("target = %q\n", cfg.Target))
	builder.WriteString("\n# Colors\n")
	builder.WriteString("# Colors support named, numeric, or hex values (ex: 7, 13, \"#ff8800\").\n")
	builder.WriteString(fmt.Sprintf("primary = %q\n", cfg.Primary))
	builder.WriteString(fmt.Sprintf("secondary = %q\n", cfg.Secondary))
	builder.WriteString(fmt.Sprintf("accent = %q\n", cfg.Accent))
	builder.WriteString(fmt.Sprintf("base_prompt = %q\n", cfg.BasePrompt))
	builder.WriteString(fmt.Sprintf("border = %q\n", cfg.Border))
	builder.WriteString("\n# CLI behavior\n")
	builder.WriteString(fmt.Sprintf("interactive_default = %t\n", cfg.InteractiveDefault))
	builder.WriteString(fmt.Sprintf("include_builtin_shorthand = %t\n", cfg.IncludeBuiltinShorthand))
	builder.WriteString("\n# History\n")
	builder.WriteString("# history_clear_cycle options: never, 1, 7, 31 (days)\n")
	builder.WriteString(fmt.Sprintf("history_clear_cycle = %q\n", cfg.HistoryClearCycle))
	builder.WriteString(fmt.Sprintf("history_file_format = %q\n", cfg.HistoryFileFormat))
	builder.WriteString(fmt.Sprintf("history_enable_time_ago = %t\n", cfg.HistoryEnableTimeAgo))
	builder.WriteString("# history_date_time options: day, month; month, day; iso; or a Go time layout\n")
	builder.WriteString(fmt.Sprintf("history_date_time = %q\n", cfg.HistoryDateTime))
	builder.WriteString(fmt.Sprintf("disable_history = %t\n", cfg.DisableHistory))
	builder.WriteString("\n[remap_short_flags]\n")
	builder.WriteString("# Map long flags to a custom short flag (single letter).\n")
	builder.WriteString("# Existing short flags: A B D E F I T V Y\n")
	builder.WriteString("# Example: directory = \"D\"\n")
	return builder.String()
}
