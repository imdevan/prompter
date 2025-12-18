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
directories, and captured command output.`,
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

func init() {
	// Add version subcommand
	rootCmd.AddCommand(versionCmd)

	// Global flags
	rootCmd.PersistentFlags().String("config", "", "config file path (default ~/.config/prompter/config.toml)")
	rootCmd.PersistentFlags().Bool("yes", false, "noninteractive mode - use defaults without prompts")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "print version information")

	// Main command flags
	rootCmd.Flags().String("pre", "", "pre-template name")
	rootCmd.Flags().String("post", "", "post-template name")
	rootCmd.Flags().StringSlice("file", []string{}, "files to include")
	rootCmd.Flags().BoolP("directory", "d", false, "include current directory")
	rootCmd.Flags().String("target", "", "output target (clipboard, stdout, file:/path)")
	rootCmd.Flags().String("editor", "", "editor to open prompt in")
	rootCmd.Flags().Bool("fix", false, "fix mode - process captured command output")
	rootCmd.Flags().String("fix-file", "", "file containing command output to fix (overrides config)")
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

	if request.Interactive, err = cmd.Flags().GetBool("yes"); err != nil {
		return nil, fmt.Errorf("invalid yes flag: %w", err)
	}
	// Invert the yes flag - yes means noninteractive
	request.Interactive = !request.Interactive

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

	return request, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

