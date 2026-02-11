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
	"prompter-cli/internal/flags"
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
	content := renderTemplateGroups(groups, theme, cfg)
	_, err = fmt.Fprintln(cmd.OutOrStdout(), content)
	return err
}

func renderTemplateGroups(groups []workflow.TemplateGroup, theme ui.Theme, cfg domain.Config) string {
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
		heading := group.Heading
		if strings.TrimSpace(heading) == "" {
			heading = fmt.Sprintf("%s templates", group.Label)
		}
		builder.WriteString(groupStyle.Render(heading))
		if strings.TrimSpace(group.Location) != "" {
			builder.WriteString("\n")
			builder.WriteString(pathStyle.Render(group.Location))
		}
		builder.WriteString("\n\n")
		builder.WriteString(renderTemplateList(group.Templates, descStyle, theme, cfg))
	}

	return builder.String()
}

func renderTemplateList(templates []domain.Template, descStyle lipgloss.Style, theme ui.Theme, cfg domain.Config) string {
	if len(templates) == 0 {
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("No templates found.")
	}
	flagLabels := listFlagLabels(templates, cfg)
	items := make([]list.Item, 0, len(templates))
	maxDescLines := 0
	for _, tmpl := range templates {
		flagLabel := flagLabels[tmpl.Name]
		items = append(items, templateListItem{
			display:     tmpl.DisplayLabel(),
			flagLabel:   flagLabel,
			description: strings.TrimSpace(tmpl.Description),
			pinned:      tmpl.Pinned,
			theme:       theme,
		})
		descLines := 0
		if strings.TrimSpace(tmpl.Description) != "" {
			descLines++
		}
		if strings.TrimSpace(flagLabel) != "" {
			descLines++
		}
		if descLines > maxDescLines {
			maxDescLines = descLines
		}
	}
	delegate := list.NewDefaultDelegate()
	if maxDescLines < 1 {
		maxDescLines = 1
	}
	delegate.SetHeight(maxDescLines + 1)
	delegate.Styles.SelectedTitle = delegate.Styles.NormalTitle
	delegate.Styles.SelectedDesc = delegate.Styles.NormalDesc
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(theme.Secondary).Bold(true)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(theme.Text)

	model := ui.NewListModel(items, delegate, 80, len(items)+2, theme)
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
	if strings.HasPrefix(strings.ToLower(t.description), "from ") {
		descStyle = lipgloss.NewStyle().Foreground(t.theme.Muted)
	}
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

func listFlagLabels(templates []domain.Template, cfg domain.Config) map[string]string {
	usedShort := flags.BuiltinShortFlags(cfg)
	labels := make(map[string]string, len(templates))
	for _, tmpl := range templates {
		if isSkillTemplateName(tmpl.Name) {
			continue
		}
		info := flags.TemplateFlags(cfg, []domain.Template{tmpl}, usedShort)
		entry, ok := info[tmpl.Name]
		if !ok {
			continue
		}
		parts := []string{}
		if entry.Shorthand != "" {
			parts = append(parts, "-"+entry.Shorthand)
		}
		if entry.Flag != "" {
			parts = append(parts, "--"+entry.Flag)
		}
		if len(parts) > 0 {
			labels[tmpl.Name] = strings.Join(parts, ", ")
		}
	}
	return labels
}

func isSkillTemplateName(name string) bool {
	path := strings.ToLower(filepath.ToSlash(strings.TrimSpace(name)))
	return strings.Contains(path, "/skills/")
}
