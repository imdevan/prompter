package main

import (
	"bytes"
	"errors"
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
	"prompter-cli/internal/ui"
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
	usedInteractive := shouldInteractive(opts, cfg)
	if usedInteractive {
		allTemplates, err := repo.List()
		if err != nil {
			return err
		}
		agentTemplates, err := workflow.AgentTemplatesForSelection(cwd, cfg, opts.agents)
		if err != nil {
			return err
		}
		if len(agentTemplates) > 0 {
			allTemplates = append(allTemplates, agentTemplates...)
		}
		prompter := interactive.New(bubbletea.NewAdapter(cfg))
		note := ""
		if opts.clipboard {
			note = "Clipboard content will be appended."
		}
		req, err := prompter.Collect(basePrompt, allTemplates, templates, cfg.ShowBasePromptInput, note)
		if err != nil {
			if errors.Is(err, interactive.ErrCanceled) {
				return printExitMessage(cmd.OutOrStdout(), cfg, "Canceled.", true)
			}
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

	target := strings.TrimSpace(opts.target)
	if target == "" {
		target = cfg.Target
	}
	editorCommand := editor.ResolveCommand(cfg.Editor)
	editorIsVim := editor.IsVim(editorCommand)
	editorTarget := opts.editorTarget && strings.EqualFold(target, "clipboard")
	if opts.editorTarget && !editorTarget {
		target = "editor"
	}

	req := domain.Request{
		BasePrompt:        basePrompt,
		TemplateNames:     templates,
		TemplateOrder:     buildTemplateOrder(templates, baseIndex),
		HistoryTag:        strings.TrimSpace(opts.historyTag),
		Files:             opts.files,
		IncludeDirectory:  opts.includeDir,
		IncludeDirPath:    opts.includeDirPath,
		DirectoryStrategy: cfg.DirectoryStrategy,
		Target:            target,
		EditorTarget:      editorTarget,
		EditorIsVim:       editorIsVim,
		PipedInput:        piped,
	}
	if opts.agents {
		req.TemplateNames = append(req.TemplateNames, "agents.md")
	}
	req.TemplateOrder, req.TemplateNames = dedupeTemplateOrder(req.TemplateOrder, req.TemplateNames)
	shorthands := templateShorthandsForNames(req.TemplateNames, opts.templateShortByName, opts.agents || containsTemplateName(req.TemplateNames, "agents.md"))
	req.HistorySuffix = buildHistorySuffix(shorthands)

	if usedInteractive && !hasPromptInput(req) {
		return printExitMessage(cmd.OutOrStdout(), cfg, "Canceled: No prompt.", true)
	}

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

func templateShorthandsForNames(names []string, mapping map[string]string, includeAgents bool) []string {
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
		if includeAgents && (key == "agents.md" || key == "agents") {
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

func dedupeTemplateOrder(order []string, names []string) ([]string, []string) {
	if len(order) == 0 {
		order = buildTemplateOrder(names, -1)
	}
	seen := make(map[string]bool)
	dedupedOrder := make([]string, 0, len(order))
	dedupedNames := make([]string, 0, len(names))
	for _, entry := range order {
		if entry == domain.BasePromptToken {
			dedupedOrder = append(dedupedOrder, entry)
			continue
		}
		key := strings.TrimSpace(strings.ToLower(entry))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		dedupedOrder = append(dedupedOrder, entry)
		dedupedNames = append(dedupedNames, entry)
	}
	return dedupedOrder, dedupedNames
}

func hasPromptInput(req domain.Request) bool {
	if strings.TrimSpace(req.BasePrompt) != "" {
		return true
	}
	if len(req.TemplateNames) > 0 {
		return true
	}
	if len(req.Files) > 0 {
		return true
	}
	if req.IncludeDirectory || req.IncludeDirPath {
		return true
	}
	if strings.TrimSpace(req.PipedInput) != "" {
		return true
	}
	return false
}

func printExitMessage(out io.Writer, cfg domain.Config, message string, mutedText bool) error {
	if out == nil {
		return nil
	}
	theme := ui.ThemeFromConfig(cfg)
	_, err := io.WriteString(out, ui.ExitMessage(theme, message, mutedText))
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, "\n")
	return err
}

func containsTemplateName(names []string, match string) bool {
	for _, name := range names {
		if strings.EqualFold(strings.TrimSpace(name), match) {
			return true
		}
	}
	return false
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
