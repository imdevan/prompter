package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/config"
	"prompter-cli/internal/domain"
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

	content := renderTemplateGroups(groups)
	_, err = fmt.Fprintln(cmd.OutOrStdout(), content)
	return err
}

func renderTemplateGroups(groups []workflow.TemplateGroup) string {
	var builder strings.Builder
	if len(groups) == 0 {
		builder.WriteString("No template locations configured.")
		return builder.String()
	}

	groupStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

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
		builder.WriteString(renderTemplateList(group.Templates, descStyle))
	}

	return builder.String()
}

func renderTemplateList(templates []domain.Template, descStyle lipgloss.Style) string {
	if len(templates) == 0 {
		return descStyle.Render("No templates found.")
	}
	items := make([]list.Item, 0, len(templates))
	for _, tmpl := range templates {
		items = append(items, templateListItem{template: tmpl})
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.NormalTitle
	delegate.Styles.SelectedDesc = delegate.Styles.NormalDesc
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("5")).Bold(true)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(lipgloss.Color("7"))

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
	template domain.Template
}

func (t templateListItem) Title() string {
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	title := strings.TrimSpace(t.template.Title)
	name := strings.TrimSpace(t.template.Name)
	display := name
	if title != "" {
		display = title
	}
	parts := []string{nameStyle.Render(display)}
	if title != "" && name != "" && isAgentTemplateName(name) {
		parts = append(parts, metaStyle.Render("("+name+")"))
	}
	flags := []string{}
	if strings.TrimSpace(t.template.Flag) != "" {
		flags = append(flags, "--"+t.template.Flag)
	}
	if strings.TrimSpace(t.template.Shorthand) != "" {
		flags = append(flags, "-"+t.template.Shorthand)
	}
	if len(flags) > 0 {
		parts = append(parts, metaStyle.Render("["+strings.Join(flags, ", ")+"]"))
	}
	if t.template.Pinned {
		parts = append(parts, metaStyle.Render("[pinned]"))
	}
	return strings.Join(parts, " ")
}

func (t templateListItem) Description() string {
	return strings.TrimSpace(t.template.Description)
}

func (t templateListItem) FilterValue() string {
	return t.template.Name
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
