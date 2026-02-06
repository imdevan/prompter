package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/config"
	"prompter-cli/internal/domain"
	"prompter-cli/internal/ui"
	"prompter-cli/internal/workflow"
)

func newListCmd() *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.includeAgents, "agents", "a", false, "include agent templates")
	return cmd
}

type listOptions struct {
	includeAgents bool
}

func runList(cmd *cobra.Command, opts *listOptions) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	manager := config.NewManager(cwd)
	cfg, err := manager.Load()
	if err != nil {
		return err
	}

	groups, err := workflow.ListTemplates(cwd, cfg, workflow.ListOptions{
		IncludeAgents: opts.includeAgents,
	})
	if err != nil {
		return err
	}

	theme := ui.ThemeFromConfig(cfg)
	content := renderTemplateGroups(groups, theme)
	_, err = fmt.Fprintln(cmd.OutOrStdout(), content)
	return err
}

func renderTemplateGroups(groups []workflow.TemplateGroup, theme ui.Theme) string {
	var builder strings.Builder
	if len(groups) == 0 {
		builder.WriteString("No template locations configured.")
		return builder.String()
	}

	groupStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Headings)
	pathStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	descStyle := lipgloss.NewStyle().Foreground(theme.Text)

	for i, group := range groups {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(groupStyle.Render(fmt.Sprintf("%s templates", group.Label)))
		if strings.TrimSpace(group.Location) != "" {
			builder.WriteString("\n")
			builder.WriteString(pathStyle.Render(group.Location))
		}
		builder.WriteString("\n\n")
		builder.WriteString(renderTemplateList(group.Templates, descStyle, theme))
	}

	return builder.String()
}

func renderTemplateList(templates []domain.Template, descStyle lipgloss.Style, theme ui.Theme) string {
	if len(templates) == 0 {
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("No templates found.")
	}
	items := make([]list.Item, 0, len(templates))
	for _, tmpl := range templates {
		items = append(items, templateListItem{
			display:     templateDisplayName(tmpl),
			flagLabel:   formatFlagLabel(tmpl),
			description: strings.TrimSpace(tmpl.Description),
			pinned:      tmpl.Pinned,
			theme:       theme,
		})
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.NormalTitle
	delegate.Styles.SelectedDesc = delegate.Styles.NormalDesc
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(theme.Secondary).Bold(true)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(theme.Text)

	model := list.New(items, delegate, 80, len(items)+2)
	model.SetShowStatusBar(false)
	model.SetShowHelp(false)
	model.SetShowPagination(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(false)
	model.Title = ""
	model.Select(0)

	var builder strings.Builder
	for i, item := range items {
		if i > 0 {
			builder.WriteString("\n")
		}
		delegate.Render(&builder, model, i, item)
		if entry, ok := item.(templateListItem); ok {
			if strings.TrimSpace(entry.Description()) != "" && i < len(items)-1 {
				builder.WriteString("\n")
			}
		}
	}
	return strings.TrimRight(builder.String(), "\n")
}

type templateListItem struct {
	display     string
	flagLabel   string
	description string
	pinned      bool
	theme       ui.Theme
}

func (t templateListItem) Title() string {
	nameStyle := lipgloss.NewStyle().Foreground(t.theme.Secondary).Bold(true)
	flagStyle := lipgloss.NewStyle().Foreground(t.theme.Tags)
	parts := []string{nameStyle.Render(t.display)}
	if t.pinned {
		parts = append(parts, flagStyle.Render("[pinned]"))
	}
	return strings.Join(parts, "  ")
}

func (t templateListItem) Description() string {
	descStyle := lipgloss.NewStyle().Foreground(t.theme.Text)
	flagStyle := lipgloss.NewStyle().Foreground(t.theme.Flags)
	parts := []string{}
	if t.description != "" {
		parts = append(parts, descStyle.Render(t.description))
	}
	if strings.TrimSpace(t.flagLabel) != "" {
		parts = append(parts, flagStyle.Render(t.flagLabel))
	}
	return strings.Join(parts, "\n")
}

func (t templateListItem) FilterValue() string {
	return t.display
}

func isAgentTemplateName(name string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "agents.md" || trimmed == "agents" {
		return true
	}
	if strings.HasPrefix(trimmed, "cursor/commands") {
		return true
	}
	if strings.HasPrefix(trimmed, "kiro/steering") {
		return true
	}
	if strings.HasPrefix(trimmed, "opencode/commands") {
		return true
	}
	return false
}

func templateDisplayName(tmpl domain.Template) string {
	title := strings.TrimSpace(tmpl.Title)
	if title != "" {
		title = toTitleCase(title)
	}
	name := strings.TrimSpace(tmpl.Name)
	if title == "" {
		return name
	}
	if name != "" && isAgentTemplateName(name) {
		return fmt.Sprintf("%s (%s)", title, name)
	}
	return title
}

func formatFlagLabel(tmpl domain.Template) string {
	flags := []string{}
	shorthand := strings.TrimSpace(tmpl.Shorthand)
	if shorthand == "" {
		shorthand = listTemplateShorthand(tmpl.Name)
	}
	flag := strings.TrimSpace(tmpl.Flag)
	if flag == "" {
		flag = listTemplateFlagName(tmpl.Name)
	}
	if shorthand != "" {
		flags = append(flags, "-"+shorthand)
	}
	if flag != "" {
		flags = append(flags, "--"+flag)
	}
	if len(flags) == 0 {
		return ""
	}
	return strings.Join(flags, ", ")
}

func toTitleCase(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		for j := 1; j < len(runes); j++ {
			runes[j] = []rune(strings.ToLower(string(runes[j])))[0]
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func listTemplateFlagName(name string) string {
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
	return strings.Trim(builder.String(), "-")
}

func listTemplateShorthand(name string) string {
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
		return string(r)
	}
	return ""
}
