package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"prompter-cli/internal/adapters/bubbletea"
	"prompter-cli/internal/adapters/clipboard"
	"prompter-cli/internal/adapters/editor"
	"prompter-cli/internal/config"
	"prompter-cli/internal/domain"
	"prompter-cli/internal/interactive"
	"prompter-cli/internal/output"
	"prompter-cli/internal/template"
	"prompter-cli/internal/workflow"

	"github.com/spf13/cobra"
)

func runGenerate(cmd *cobra.Command, opts *rootOptions, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	manager := config.NewManager(cwd)
	cfg, err := manager.LoadWithOverride(opts.configPath)
	if err != nil {
		return err
	}

	basePrompt := strings.TrimSpace(strings.Join(args, " "))
	clip := clipboard.Adapter{}
	clipText := ""
	if opts.clipboard {
		text, err := clip.ReadText()
		if err != nil {
			return err
		}
		clipText = strings.TrimSpace(text)
	}

	piped, err := readStdinIfPiped(os.Stdin)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.fixOutput) != "" {
		piped = opts.fixOutput
	}

	localPrompts := ""
	if cfg.LocalPromptsLocation != "" {
		localPrompts = filepath.Join(cwd, cfg.LocalPromptsLocation)
	}
	repo := template.NewRepository(localPrompts, cfg.PromptsLocation)

	argv := os.Args[1:]
	if cmd.Name() != "prompter" {
		argv = stripSubcommand(argv, cmd.Name())
	}
	orderedTemplates, baseIndex, _ := resolveTemplateOrder(argv, opts, cfg)
	templates := orderedTemplates
	if len(templates) == 0 {
		templates = append([]string{}, opts.templates...)
	}
	if shouldInteractive(opts, cfg) {
		allTemplates, err := repo.List()
		if err != nil {
			return err
		}
		if agentTemplate, err := agentTemplateForSelection(cwd); err != nil {
			return err
		} else if agentTemplate != nil {
			allTemplates = append(allTemplates, *agentTemplate)
		}
		prompter := interactive.New(bubbletea.Adapter{})
		note := ""
		if opts.clipboard {
			note = "Clipboard content will be appended."
		}
		req, err := prompter.Collect(basePrompt, allTemplates, templates, opts.clipboard, note)
		if err != nil {
			return err
		}
		basePrompt = req.BasePrompt
		templates = req.TemplateNames
	}

	if opts.clipboard && clipText != "" {
		if basePrompt == "" {
			basePrompt = clipText
		} else {
			basePrompt = strings.TrimSpace(basePrompt + "\n\n" + clipText)
		}
	}

	req := domain.Request{
		BasePrompt:        basePrompt,
		TemplateNames:     templates,
		TemplateOrder:     buildTemplateOrder(templates, baseIndex),
		Files:             opts.files,
		IncludeDirectory:  opts.includeDir,
		DirectoryStrategy: cfg.DirectoryStrategy,
		Target:            opts.target,
		PipedInput:        piped,
	}

	if opts.fix {
		req.Fix.Enabled = true
		req.Fix.Output = piped
	}
	if opts.agents {
		req.TemplateNames = append(req.TemplateNames, "agents.md")
	}
	req.TemplateNames = dedupeTemplates(req.TemplateNames)
	shorthands := templateShorthandsForNames(req.TemplateNames, opts.templateShortByName)
	req.HistorySuffix = buildHistorySuffix(shorthands)

	handler := output.NewHandler(cmd.OutOrStdout(), clip, editor.New(cfg.Editor))
	service := workflow.New(repo, handler)
	_, err = service.Generate(req, cfg)
	return err
}

func buildTemplateOrder(templates []string, baseIndex int) []string {
	if baseIndex < 0 || baseIndex > len(templates) {
		baseIndex = len(templates)
	}
	order := make([]string, 0, len(templates)+1)
	for i, name := range templates {
		if i == baseIndex {
			order = append(order, domain.BasePromptToken)
		}
		order = append(order, name)
	}
	if baseIndex == len(templates) {
		order = append(order, domain.BasePromptToken)
	}
	return order
}

func templateShorthandsForNames(names []string, mapping map[string]string) []string {
	if len(names) == 0 || len(mapping) == 0 {
		return nil
	}
	shorts := make([]string, 0, len(names))
	seen := make(map[string]bool)
	for _, name := range names {
		key := strings.TrimSpace(strings.ToLower(name))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if key == "agents.md" || key == "agents" {
			shorts = append(shorts, "a")
			continue
		}
		if shorthand := strings.TrimSpace(mapping[name]); shorthand != "" {
			shorts = append(shorts, shorthand)
		}
	}
	return shorts
}

func buildHistorySuffix(shorts []string) string {
	if len(shorts) == 0 {
		return ""
	}
	return strings.Join(shorts, "")
}

func dedupeTemplates(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	unique := make([]string, 0, len(names))
	for _, name := range names {
		key := strings.TrimSpace(strings.ToLower(name))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, name)
	}
	return unique
}

func stripSubcommand(args []string, name string) []string {
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if arg == name {
			return append(append([]string{}, args[:i]...), args[i+1:]...)
		}
	}
	return args
}

func agentTemplateForSelection(cwd string) (*domain.Template, error) {
	path := filepath.Join(cwd, "AGENTS.md")
	if _, err := os.Stat(path); err == nil {
		return &domain.Template{
			Name:        "agents.md",
			Title:       "Agent instructions",
			Description: "From AGENTS.md",
		}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return nil, nil
}

func shouldInteractive(opts *rootOptions, cfg domain.Config) bool {
	if opts.yes {
		return false
	}
	if opts.interactive {
		return true
	}
	return cfg.InteractiveDefault
}

func readStdinIfPiped(r *os.File) (string, error) {
	info, err := r.Stat()
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
