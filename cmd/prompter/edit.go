package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/adapters/editor"
	"prompter-cli/internal/config"
	"prompter-cli/internal/ui"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit an existing template",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEdit(cmd, args)
		},
	}
}

func runEdit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	manager := config.NewManager(cwd)
	cfg, err := manager.Load()
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.PromptsLocation) == "" {
		return errors.New("prompts_location is not configured")
	}

	editorAdapter := editor.New(cfg.Editor)
	if len(args) == 0 {
		if err := editorAdapter.Open(cfg.PromptsLocation); err != nil {
			return err
		}
		cmd.Printf("Opened prompts directory %s\n", cfg.PromptsLocation)
		return nil
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return errors.New("template name is required")
	}
	name = strings.TrimSuffix(name, ".md")
	templatePath := filepath.Join(cfg.PromptsLocation, name+".md")
	if _, err := os.Stat(templatePath); err == nil {
		if err := editorAdapter.Open(templatePath); err != nil {
			return err
		}
		cmd.Printf("Opened template %s\n", name)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	theme := ui.ThemeFromConfig(cfg)
	confirm, err := promptCreateTemplate(name, theme)
	if err != nil {
		return err
	}
	if !confirm {
		cmd.Printf("Template %s not created\n", name)
		return nil
	}
	content := defaultTemplateContent(name)
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(templatePath, []byte(content), 0o644); err != nil {
		return err
	}
	if err := editorAdapter.Open(templatePath); err != nil {
		return err
	}
	cmd.Printf("Created template %s\n", name)
	return nil
}

func defaultTemplateContent(name string) string {
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = "New template"
	}
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString(fmt.Sprintf("name: %q\n", displayName))
	builder.WriteString("description: \"Describe when to use this template\"\n")
	builder.WriteString("flag: \"\"\n")
	builder.WriteString("shorthand: \"\"\n")
	builder.WriteString("pin: false\n")
	builder.WriteString("---\n\n")
	builder.WriteString("# ")
	builder.WriteString(displayName)
	builder.WriteString("\n\n")
	builder.WriteString("Write your template here.\n")
	return builder.String()
}

type confirmModel struct {
	title   string
	prompt  string
	choice  *bool
	confirm *huh.Confirm
	theme   ui.Theme
}

func promptCreateTemplate(name string, theme ui.Theme) (bool, error) {
	model := newConfirmModel("Create template?", fmt.Sprintf("Template %q not found. Create it? (y/n)", name), theme)
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return false, err
	}
	if m, ok := result.(confirmModel); ok {
		return m.choiceValue(), nil
	}
	return false, fmt.Errorf("unexpected model result")
}

func (m confirmModel) Init() tea.Cmd {
	if m.confirm != nil {
		return m.confirm.Init()
	}
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.setChoice(false)
			return m, tea.Quit
		case tea.KeyEnter:
			return m, tea.Quit
		}
		switch msg.String() {
		case "y", "Y":
			m.setChoice(true)
			return m, tea.Quit
		case "n", "N":
			m.setChoice(false)
			return m, tea.Quit
		}
	}
	if m.confirm == nil {
		return m, nil
	}
	updated, cmd := m.confirm.Update(msg)
	if confirm, ok := updated.(*huh.Confirm); ok {
		m.confirm = confirm
	}
	return m, cmd
}

func (m confirmModel) View() string {
	titleText := m.title
	if strings.TrimSpace(titleText) == "" {
		titleText = "Confirm"
	}
	helpText := "Press y or n."
	title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Headings).Render(titleText)
	prompt := lipgloss.NewStyle().Foreground(m.theme.Text).Render(m.prompt)
	help := lipgloss.NewStyle().Foreground(m.theme.Muted).Render(helpText)
	confirmView := ""
	if m.confirm != nil {
		confirmView = m.confirm.View()
	}
	content := strings.Join([]string{
		title,
		"",
		confirmView,
		"",
		prompt,
		help,
	}, "\n")
	return lipgloss.NewStyle().
		Margin(1, 1).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Render(content)
}

func newConfirmModel(title, prompt string, theme ui.Theme) confirmModel {
	choice := false
	helpText := "Press y or n."
	width := confirmDialogWidth(title, prompt, helpText)
	confirm := huh.NewConfirm().
		Title("").
		Description("").
		Value(&choice)
	confirm.WithKeyMap(huh.NewDefaultKeyMap())
	confirm.WithWidth(width)
	confirm.WithButtonAlignment(lipgloss.Center)
	confirm.WithTheme(confirmHuhTheme(theme))
	confirm.Focus()
	return confirmModel{
		title:   title,
		prompt:  prompt,
		choice:  &choice,
		confirm: confirm,
		theme:   theme,
	}
}

func confirmDialogWidth(title, prompt, help string) int {
	width := lipgloss.Width(title)
	if promptWidth := lipgloss.Width(prompt); promptWidth > width {
		width = promptWidth
	}
	if helpWidth := lipgloss.Width(help); helpWidth > width {
		width = helpWidth
	}
	const minWidth = 32
	if width < minWidth {
		width = minWidth
	}
	return width
}

func confirmHuhTheme(theme ui.Theme) *huh.Theme {
	huhTheme := huh.ThemeBase()
	huhTheme.Focused.Base = lipgloss.NewStyle()
	huhTheme.Blurred.Base = lipgloss.NewStyle()
	huhTheme.Focused.FocusedButton = lipgloss.NewStyle().
		Padding(0, 2).
		MarginRight(1).
		Foreground(theme.Text).
		Background(theme.Border).
		Bold(true)
	huhTheme.Focused.BlurredButton = lipgloss.NewStyle().
		Padding(0, 2).
		MarginRight(1).
		Foreground(theme.Text)
	huhTheme.Blurred.FocusedButton = huhTheme.Focused.FocusedButton
	huhTheme.Blurred.BlurredButton = huhTheme.Focused.BlurredButton
	return huhTheme
}

func (m confirmModel) setChoice(value bool) {
	if m.choice == nil {
		return
	}
	*m.choice = value
}

func (m confirmModel) choiceValue() bool {
	if m.choice == nil {
		return false
	}
	return *m.choice
}
