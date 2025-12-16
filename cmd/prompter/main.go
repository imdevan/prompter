package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"prompter-cli/internal/app"
)

var rootCmd = &cobra.Command{
	Use:   "prompter",
	Short: "A CLI tool for assembling AI coding prompts",
	Long: `Prompter CLI assembles high-quality prompts for AI coding agents by combining 
base prompts with optional pre/post templates and contextual information from files, 
directories, and captured command output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.Run(cmd, args)
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().String("config", "", "config file path (default ~/.config/prompter/config.toml)")
	rootCmd.PersistentFlags().Bool("yes", false, "noninteractive mode - use defaults without prompts")
	
	// Main command flags
	rootCmd.Flags().String("pre", "", "pre-template name")
	rootCmd.Flags().String("post", "", "post-template name")
	rootCmd.Flags().StringSlice("file", []string{}, "files to include")
	rootCmd.Flags().String("directory", "", "directory to include")
	rootCmd.Flags().String("target", "", "output target (clipboard, stdout, file:/path)")
	rootCmd.Flags().String("editor", "", "editor to open prompt in")
	rootCmd.Flags().Bool("fix", false, "fix mode - process captured command output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}