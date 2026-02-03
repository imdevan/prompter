package main

import "github.com/spf13/cobra"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type rootOptions struct {
	configPath   string
	files        []string
	includeDir   bool
	target       string
	fix          bool
	clipboard    bool
	agents       bool
	interactive  bool
	yes          bool
	editorTarget bool
	templates    []string
	showVersion  bool
	fixOutput    string
}

var rootCmd = newRootCmd()

// Execute is the CLI entrypoint.
func Execute() error {
	return rootCmd.Execute()
}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{}
	cmd := &cobra.Command{
		Use:   "prompter [base-prompt]",
		Short: "Assemble prompts for AI coding agents",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.showVersion {
				cmd.Printf("version=%s commit=%s date=%s\n", version, commit, date)
				return nil
			}
			if opts.editorTarget {
				opts.target = "editor"
			}
			return runGenerate(cmd, opts, args)
		},
	}

	cmd.Flags().StringVar(&opts.configPath, "config", "", "config file path")
	cmd.Flags().StringSliceVar(&opts.files, "file", nil, "files to include")
	cmd.Flags().BoolVarP(&opts.includeDir, "directory", "d", false, "include current directory")
	cmd.Flags().StringVarP(&opts.target, "target", "t", "", "output target (clipboard, stdout, file:/path, editor)")
	cmd.Flags().BoolVarP(&opts.fix, "fix", "f", false, "fix mode")
	cmd.Flags().BoolVarP(&opts.clipboard, "clipboard", "b", false, "use clipboard input")
	cmd.Flags().BoolVarP(&opts.agents, "agents", "a", false, "include AGENTS.md/.cursor/.kiro templates")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "force interactive mode")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "non-interactive mode")
	cmd.Flags().BoolVarP(&opts.editorTarget, "editor", "e", false, "open output in editor")
	cmd.Flags().StringSliceVar(&opts.templates, "template", nil, "template names to include")
	cmd.Flags().BoolVarP(&opts.showVersion, "version", "v", false, "print version information")

	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newEditCmd())
	cmd.AddCommand(newFixCmd())

	return cmd
}
