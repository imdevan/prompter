package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"prompter-cli/internal/config"
	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type rootOptions struct {
	configPath            string
	files                 []string
	includeDir            bool
	target                string
	fix                   bool
	clipboard             bool
	agents                bool
	interactive           bool
	yes                   bool
	editorTarget          bool
	templates             []string
	showVersion           bool
	fixOutput             string
	historyTag            string
	templateFlagName      map[string]string
	templateFlagShorthand map[string]string
	templateShortByName   map[string]string
}

var rootCmd = newRootCmd()

// Execute is the CLI entrypoint.
func Execute() error {
	return rootCmd.Execute()
}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{}
	cfg := loadConfigForFlagRegistration()
	cmd := &cobra.Command{
		Use:   "prompter [base-prompt]",
		Short: "Assemble prompts for AI coding agents",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.showVersion {
				cmd.Printf("version=%s commit=%s date=%s\n", version, commit, date)
				return nil
			}
			return runGenerate(cmd, opts, args)
		},
	}

	addRootFlags(cmd, opts, cfg, rootFlagOptions{
		includeFix:     true,
		includeVersion: true,
	})
	registerTemplateFlags(cmd, opts, cfg, nil)
	setTemplateHelp(cmd)

	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newEditCmd())
	cmd.AddCommand(newFixCmd())
	cmd.AddCommand(newHistoryCmd())
	cmd.AddCommand(newCompletionCmd())

	return cmd
}

type rootFlagOptions struct {
	includeFix     bool
	includeVersion bool
}

func addRootFlags(cmd *cobra.Command, opts *rootOptions, cfg domain.Config, options rootFlagOptions) {
	addString := func(longName, defaultShort, usage string, target *string) {
		if shorthand := builtinShortFlag(cfg, longName, defaultShort); shorthand != "" {
			cmd.Flags().StringVarP(target, longName, shorthand, "", usage)
		} else {
			cmd.Flags().StringVar(target, longName, "", usage)
		}
	}
	addBool := func(longName, defaultShort, usage string, target *bool) {
		if shorthand := builtinShortFlag(cfg, longName, defaultShort); shorthand != "" {
			cmd.Flags().BoolVarP(target, longName, shorthand, false, usage)
		} else {
			cmd.Flags().BoolVar(target, longName, false, usage)
		}
	}
	addStringSlice := func(longName, defaultShort, usage string, target *[]string) {
		if shorthand := builtinShortFlag(cfg, longName, defaultShort); shorthand != "" {
			cmd.Flags().StringSliceVarP(target, longName, shorthand, nil, usage)
		} else {
			cmd.Flags().StringSliceVar(target, longName, nil, usage)
		}
	}

	addString("config", "", "config file path", &opts.configPath)
	addStringSlice("file", "", "files to include", &opts.files)
	addBool("directory", "D", "include current directory", &opts.includeDir)
	addString("target", "T", "output target (clipboard, stdout, file:/path, editor)", &opts.target)
	if options.includeFix {
		addBool("fix", "F", "fix mode", &opts.fix)
	}
	addBool("clipboard", "B", "use clipboard input", &opts.clipboard)
	addBool("agents", "A", "include AGENTS.md/.cursor/.kiro templates", &opts.agents)
	addBool("interactive", "I", "force interactive mode", &opts.interactive)
	addBool("yes", "Y", "non-interactive mode", &opts.yes)
	addBool("editor", "E", "open output in editor", &opts.editorTarget)
	addStringSlice("template", "", "template names to include", &opts.templates)
	addString("tag", "G", "tag to include in history frontmatter", &opts.historyTag)
	if options.includeVersion {
		addBool("version", "V", "print version information", &opts.showVersion)
	}
}

func registerTemplateFlags(cmd *cobra.Command, opts *rootOptions, cfg domain.Config, templates []domain.Template) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	localPrompts := ""
	if cfg.LocalPromptsLocation != "" {
		localPrompts = filepath.Join(cwd, cfg.LocalPromptsLocation)
	}
	if templates == nil {
		repo := template.NewRepository(localPrompts, cfg.PromptsLocation)
		var err error
		templates, err = repo.List()
		if err != nil {
			return
		}
	}

	if opts.templateFlagName == nil {
		opts.templateFlagName = make(map[string]string)
	}
	if opts.templateFlagShorthand == nil {
		opts.templateFlagShorthand = make(map[string]string)
	}
	if opts.templateShortByName == nil {
		opts.templateShortByName = make(map[string]string)
	}
	usedShort := map[string]bool{}
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Shorthand != "" {
			usedShort[flag.Shorthand] = true
		}
	})

	for _, tmpl := range templates {
		flagName := strings.TrimSpace(tmpl.Flag)
		if flagName == "" {
			flagName = defaultTemplateFlagName(tmpl.Name)
		}
		if flagName == "" {
			continue
		}
		if cmd.Flags().Lookup(flagName) != nil {
			continue
		}
		shorthand := strings.TrimSpace(tmpl.Shorthand)
		if shorthand != "" && len([]rune(shorthand)) != 1 {
			shorthand = ""
		}
		if shorthand != "" && usedShort[shorthand] {
			shorthand = ""
		}
		if shorthand == "" {
			shorthand = templateShorthandFromName(tmpl.Name, usedShort)
		}
		usage := strings.TrimSpace(tmpl.Description)
		if usage == "" {
			usage = "include template " + tmpl.Name
		}
		if shorthand != "" {
			cmd.Flags().BoolP(flagName, shorthand, false, usage)
			usedShort[shorthand] = true
		} else {
			cmd.Flags().Bool(flagName, false, usage)
		}
		opts.templateFlagName[flagName] = tmpl.Name
		if shorthand != "" {
			opts.templateFlagShorthand[shorthand] = tmpl.Name
			opts.templateShortByName[tmpl.Name] = shorthand
		}
		if flag := cmd.Flags().Lookup(flagName); flag != nil {
			if flag.Annotations == nil {
				flag.Annotations = make(map[string][]string)
			}
			flag.Annotations["template"] = []string{"true"}
		}
	}
}

func loadConfigForFlagRegistration() domain.Config {
	cwd, err := os.Getwd()
	if err != nil {
		return domain.DefaultConfig()
	}
	manager := config.NewManager(cwd)
	configPath := extractConfigPath(os.Args[1:])
	if configPath != "" {
		if cfg, err := manager.LoadWithOverride(configPath); err == nil {
			return cfg
		}
	}
	if cfg, err := manager.Load(); err == nil {
		return cfg
	}
	return domain.DefaultConfig()
}

func extractConfigPath(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config=")
		}
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func defaultTemplateFlagName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	flagName := strings.Trim(builder.String(), "-")
	return flagName
}

func templateShorthandFromName(name string, used map[string]bool) string {
	base := strings.TrimSpace(filepath.Base(name))
	if base == "" {
		return ""
	}
	for _, r := range base {
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		if r < 'a' || r > 'z' {
			continue
		}
		letter := string(r)
		if !used[letter] {
			return letter
		}
	}
	return ""
}

func resolveTemplateOrder(args []string, opts *rootOptions, cfg domain.Config) ([]string, int, []string) {
	templates := make([]string, 0)
	shorts := make([]string, 0)
	baseIndex := -1

	longFlagsWithValue := map[string]bool{
		"config":   true,
		"file":     true,
		"target":   true,
		"template": true,
		"tag":      true,
	}
	shortFlagsWithValue := make(map[string]bool)
	templateShort := builtinShortFlag(cfg, "template", "")
	for long, def := range map[string]string{
		"config":   "",
		"file":     "",
		"target":   "T",
		"template": "",
		"tag":      "",
	} {
		if shorthand := builtinShortFlag(cfg, long, def); shorthand != "" {
			shortFlagsWithValue[shorthand] = true
		}
	}

	addTemplate := func(name string, shorthand string) {
		if name == "" {
			return
		}
		templates = append(templates, name)
		if shorthand == "" {
			shorthand = opts.templateShortByName[name]
		}
		if shorthand != "" {
			shorts = append(shorts, shorthand)
		}
	}

	addTemplatesFromValue := func(value string) {
		parts := strings.Split(value, ",")
		for _, part := range parts {
			name := strings.TrimSpace(part)
			if name == "" {
				continue
			}
			addTemplate(name, "")
		}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if baseIndex == -1 {
				baseIndex = len(templates)
			}
			break
		}
		if strings.HasPrefix(arg, "--") && len(arg) > 2 {
			nameVal := strings.TrimPrefix(arg, "--")
			name, value, hasValue := strings.Cut(nameVal, "=")
			if name == "template" {
				if hasValue {
					addTemplatesFromValue(value)
				} else if i+1 < len(args) {
					i++
					addTemplatesFromValue(args[i])
				}
				continue
			}
			if longFlagsWithValue[name] {
				if !hasValue && i+1 < len(args) {
					i++
				}
				continue
			}
			if tmplName, ok := opts.templateFlagName[name]; ok {
				addTemplate(tmplName, opts.templateShortByName[tmplName])
			}
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			shortsArg := strings.TrimPrefix(arg, "-")
			shortsArg, value, hasValue := strings.Cut(shortsArg, "=")
			for idx, r := range shortsArg {
				short := string(r)
				if tmplName, ok := opts.templateFlagShorthand[short]; ok {
					addTemplate(tmplName, short)
					continue
				}
				if short == templateShort && idx == len(shortsArg)-1 {
					if hasValue {
						addTemplatesFromValue(value)
					} else if i+1 < len(args) {
						i++
						addTemplatesFromValue(args[i])
					}
					break
				}
				if shortFlagsWithValue[short] {
					if idx == len(shortsArg)-1 && !hasValue && i+1 < len(args) {
						i++
					}
					break
				}
			}
			continue
		}
		if baseIndex == -1 {
			baseIndex = len(templates)
		}
	}

	return templates, baseIndex, shorts
}

func builtinShortFlag(cfg domain.Config, longName, defaultShort string) string {
	if cfg.RemapShortFlags != nil {
		if remapped, ok := cfg.RemapShortFlags[longName]; ok {
			remapped = strings.TrimSpace(remapped)
			if len([]rune(remapped)) == 1 {
				return remapped
			}
		}
	}
	if !cfg.IncludeBuiltinShorthand {
		return ""
	}
	return defaultShort
}

func setTemplateHelp(cmd *cobra.Command) {
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Usage:\n  %s\n", cmd.UseLine())
		if cmd.Long != "" {
			fmt.Fprintf(out, "\n%s\n", cmd.Long)
		} else if cmd.Short != "" {
			fmt.Fprintf(out, "\n%s\n", cmd.Short)
		}
		if cmd.HasAvailableSubCommands() {
			fmt.Fprintln(out, "\nCommands:")
			for _, sub := range cmd.Commands() {
				if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
					continue
				}
				fmt.Fprintf(out, "  %s\t%s\n", sub.Name(), sub.Short)
			}
		}

		builtIn, templateFlags := splitTemplateFlags(cmd)
		if builtIn.HasFlags() {
			fmt.Fprintln(out, "\nFlags:")
			fmt.Fprint(out, builtIn.FlagUsagesWrapped(80))
		}
		if templateFlags.HasFlags() {
			fmt.Fprintln(out, "\nTemplate Flags:")
			fmt.Fprint(out, templateFlags.FlagUsagesWrapped(80))
		}
	})
}

func splitTemplateFlags(cmd *cobra.Command) (*pflag.FlagSet, *pflag.FlagSet) {
	builtIn := pflag.NewFlagSet("flags", pflag.ContinueOnError)
	templateFlags := pflag.NewFlagSet("template", pflag.ContinueOnError)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		if isTemplateFlag(flag) {
			templateFlags.AddFlag(flag)
			return
		}
		builtIn.AddFlag(flag)
	})
	return builtIn, templateFlags
}

func isTemplateFlag(flag *pflag.Flag) bool {
	if flag.Annotations == nil {
		return false
	}
	values, ok := flag.Annotations["template"]
	if !ok || len(values) == 0 {
		return false
	}
	return values[0] == "true"
}
