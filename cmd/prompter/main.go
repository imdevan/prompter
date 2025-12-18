package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"prompter-cli/internal/app"
	"prompter-cli/pkg/models"
)

// Build-time variables injected via ldflags
var (
	version   = "dev"
	commit    = "unknown"
	date      = "unknown"
	goVersion = runtime.Version()
)

var rootCmd = &cobra.Command{
	Use:   "prompter [base-prompt]",
	Short: "A CLI tool for assembling AI coding prompts",
	Long: `Prompter CLI assembles high-quality prompts for AI coding agents by combining 
base prompts with optional pre/post templates and contextual information from files, 
directories, and captured command output.

The base prompt can be provided as an argument, entered interactively, or read from 
clipboard using --clipboard. When both an argument and --clipboard are provided, 
the clipboard content is appended to the base prompt.

Interactive mode can be controlled via config (interactive_default), overridden with 
-i (force interactive) or -y (force non-interactive).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if version flag is set
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			versionCmd.Run(cmd, args)
			return nil
		}

		request, err := buildRequestFromFlags(cmd, args)
		if err != nil {
			return fmt.Errorf("invalid arguments: %w", err)
		}

		return app.Run(request)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print detailed version information including build version, commit, date, and platform details.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("prompter version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built: %s\n", date)
		fmt.Printf("  go version: %s\n", goVersion)
		fmt.Printf("  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available prompt templates",
	Long:  "List all available pre and post prompt templates from the configured prompts directory.",
	RunE: func(cmd *cobra.Command, args []string) error {
		request := models.NewPromptRequest()
		
		// Get config path from flag
		if configPath, err := cmd.Flags().GetString("config"); err == nil {
			request.ConfigPath = configPath
		}
		
		return app.ListTemplates(request)
	},
}

var addCmd = &cobra.Command{
	Use:   "add [content]",
	Short: "Add a new prompt template",
	Long:  "Add a new prompt template to the configured prompts directory. Use -p for pre-templates or -o for post-templates. If no flags are provided, interactive mode will ask for template type and name.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		request := models.NewPromptRequest()
		
		// Get config path from flag
		if configPath, err := cmd.Flags().GetString("config"); err == nil {
			request.ConfigPath = configPath
		}
		
		// Handle interactive mode flags
		if forceNonInteractive, err := cmd.Flags().GetBool("yes"); err == nil {
			request.ForceNonInteractive = forceNonInteractive
		}
		
		if forceInteractive, err := cmd.Flags().GetBool("interactive"); err == nil {
			request.ForceInteractive = forceInteractive
		}
		
		// Validate that both flags are not set
		if request.ForceInteractive && request.ForceNonInteractive {
			return fmt.Errorf("cannot use both --interactive and --yes flags")
		}
		
		// Set initial interactive mode (will be resolved after config loading)
		request.Interactive = true // Default, will be overridden by config resolution
		
		// Get content from argument if provided
		var content string
		if len(args) > 0 {
			content = args[0]
		}
		
		// Get flags
		preName, _ := cmd.Flags().GetString("pre")
		postName, _ := cmd.Flags().GetString("post")
		fromClipboard, _ := cmd.Flags().GetBool("clipboard")
		
		return app.AddTemplate(request, content, preName, postName, fromClipboard)
	},
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(addCmd)
	
	// Add command specific flags
	addCmd.Flags().StringP("pre", "p", "", "create a pre-template with the specified name")
	addCmd.Flags().StringP("post", "o", "", "create a post-template with the specified name")
	addCmd.Flags().BoolP("clipboard", "b", false, "create template from clipboard content")

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path (default ~/.config/prompter/config.toml)")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "noninteractive mode - use defaults without prompts")
	rootCmd.PersistentFlags().BoolP("interactive", "i", false, "force interactive mode (overrides config default)")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "print version information")

	// Main command flags
	rootCmd.Flags().StringP("pre", "p", "", "pre-template name")
	rootCmd.Flags().StringP("post", "o", "", "post-template name")
	rootCmd.Flags().StringSlice("file", []string{}, "files to include")
	rootCmd.Flags().BoolP("directory", "d", false, "include current directory")
	rootCmd.Flags().StringP("target", "t", "", "output target (clipboard, stdout, file:/path)")
	rootCmd.Flags().StringP("editor", "e", "", "editor to open prompt in")
	rootCmd.Flags().BoolP("fix", "f", false, "fix mode - process captured command output")
	rootCmd.Flags().String("fix-file", "", "file containing command output to fix (overrides config)")
	rootCmd.Flags().BoolP("numbers", "n", false, "enable number key selection for templates")
	rootCmd.Flags().BoolP("clipboard", "b", false, "append clipboard content to prompt (or use as base prompt if none provided)")
}

// buildRequestFromFlags constructs a PromptRequest from command flags and arguments
func buildRequestFromFlags(cmd *cobra.Command, args []string) (*models.PromptRequest, error) {
	request := models.NewPromptRequest()

	// Get base prompt from positional argument
	if len(args) > 0 {
		request.BasePrompt = strings.TrimSpace(args[0])
	}

	// Extract flags
	var err error

	if request.ConfigPath, err = cmd.Flags().GetString("config"); err != nil {
		return nil, fmt.Errorf("invalid config flag: %w", err)
	}

	// Handle interactive mode flags
	if request.ForceNonInteractive, err = cmd.Flags().GetBool("yes"); err != nil {
		return nil, fmt.Errorf("invalid yes flag: %w", err)
	}
	
	if request.ForceInteractive, err = cmd.Flags().GetBool("interactive"); err != nil {
		return nil, fmt.Errorf("invalid interactive flag: %w", err)
	}
	
	// Validate that both flags are not set
	if request.ForceInteractive && request.ForceNonInteractive {
		return nil, fmt.Errorf("cannot use both --interactive and --yes flags")
	}
	
	// Set initial interactive mode (will be resolved after config loading)
	request.Interactive = true // Default, will be overridden by config resolution

	if request.PreTemplate, err = cmd.Flags().GetString("pre"); err != nil {
		return nil, fmt.Errorf("invalid pre flag: %w", err)
	}

	if request.PostTemplate, err = cmd.Flags().GetString("post"); err != nil {
		return nil, fmt.Errorf("invalid post flag: %w", err)
	}

	if request.Files, err = cmd.Flags().GetStringSlice("file"); err != nil {
		return nil, fmt.Errorf("invalid file flag: %w", err)
	}

	var includeDirectory bool
	if includeDirectory, err = cmd.Flags().GetBool("directory"); err != nil {
		return nil, fmt.Errorf("invalid directory flag: %w", err)
	}
	
	// If --directory flag is set, use current directory
	if includeDirectory {
		if cwd, err := os.Getwd(); err == nil {
			request.Directory = cwd
		} else {
			request.Directory = "."
		}
	}

	if request.Target, err = cmd.Flags().GetString("target"); err != nil {
		return nil, fmt.Errorf("invalid target flag: %w", err)
	}

	if request.Editor, err = cmd.Flags().GetString("editor"); err != nil {
		return nil, fmt.Errorf("invalid editor flag: %w", err)
	}
	// Track if --editor flag was explicitly set
	request.EditorRequested = cmd.Flags().Changed("editor")

	if request.FixMode, err = cmd.Flags().GetBool("fix"); err != nil {
		return nil, fmt.Errorf("invalid fix flag: %w", err)
	}

	// Handle fix-file flag (overrides config)
	if fixFile, err := cmd.Flags().GetString("fix-file"); err != nil {
		return nil, fmt.Errorf("invalid fix-file flag: %w", err)
	} else if fixFile != "" {
		request.FixFile = fixFile
	}

	if request.NumberSelect, err = cmd.Flags().GetBool("numbers"); err != nil {
		return nil, fmt.Errorf("invalid numbers flag: %w", err)
	}

	if request.FromClipboard, err = cmd.Flags().GetBool("clipboard"); err != nil {
		return nil, fmt.Errorf("invalid clipboard flag: %w", err)
	}



	return request, nil
}

func main() {
	// Disable usage on error to show only our custom error messages
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

