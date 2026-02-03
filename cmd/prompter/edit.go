package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/adapters/editor"
	"prompter-cli/internal/config"
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

	confirm, err := promptCreateTemplate(name)
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
	title := strings.TrimSpace(name)
	if title == "" {
		title = "New template"
	}
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString(fmt.Sprintf("title: %q\n", title))
	builder.WriteString("description: \"Describe when to use this template\"\n")
	builder.WriteString("flag: \"\"\n")
	builder.WriteString("shorthand: \"\"\n")
	builder.WriteString("pin: false\n")
	builder.WriteString("---\n\n")
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	builder.WriteString("Write your template here.\n")
	return builder.String()
}

type confirmModel struct {
	prompt  string
	choice  bool
	decided bool
}

func promptCreateTemplate(name string) (bool, error) {
	model := confirmModel{
		prompt: fmt.Sprintf("Template %q not found. Create it? (y/n)", name),
	}
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return false, err
	}
	if m, ok := result.(confirmModel); ok {
		return m.choice, nil
	}
	return false, fmt.Errorf("unexpected model result")
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.choice = true
			m.decided = true
			return m, tea.Quit
		case "n", "N", "esc", "ctrl+c":
			m.choice = false
			m.decided = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("Create template?")
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render(m.prompt)
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render("Press y or n.")
	return strings.Join([]string{title, prompt, help}, "\n")
}
