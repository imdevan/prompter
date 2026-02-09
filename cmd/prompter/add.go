package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/adapters/editor"
	"prompter-cli/internal/config"
	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
	"prompter-cli/internal/ui"
)

type addOptions struct {
	force        bool
	openInEditor bool
	interactive  bool
}

func newAddCmd() *cobra.Command {
	opts := &addOptions{}
	cmd := &cobra.Command{
		Use:   "add [name] [content]",
		Short: "Add a new template",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, opts, args)
		},
	}
	cmd.Flags().BoolVarP(&opts.openInEditor, "editor", "e", false, "open template in editor after creation")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "overwrite existing template")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "prompt for template name and content")
	return cmd
}

func runAdd(cmd *cobra.Command, opts *addOptions, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if len(args) <= 1 {
		opts.interactive = true
	}
	manager := config.NewManager(cwd)
	cfg, err := manager.Load()
	if err != nil {
		return err
	}

	name, content, err := resolveAddInputs(args)
	if err != nil {
		return err
	}

	if opts.interactive {
		theme := ui.ThemeFromConfig(cfg)
		var canceled bool
		name, content, canceled, err = promptAddTemplate(name, content, theme)
		if err != nil {
			return err
		}
		if canceled {
			return printExitMessage(cmd.OutOrStdout(), cfg, "Canceled.", true)
		}
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return printExitMessage(cmd.OutOrStdout(), cfg, "Empty template name.", true)
	}
	name = strings.TrimSuffix(name, ".md")
	if strings.TrimSpace(content) == "" {
		return printExitMessage(cmd.OutOrStdout(), cfg, "Empty template content.", true)
	}
	if cfg.PromptsLocation == "" {
		return errors.New("prompts_location is not configured")
	}

	repo := template.NewRepository(cfg.PromptsLocation)
	templatePath := filepath.Join(cfg.PromptsLocation, name+".md")
	if !opts.force {
		if _, err := os.Stat(templatePath); err == nil {
			return fmt.Errorf("template already exists: %s (use --force to overwrite)", templatePath)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	if err := repo.Save(domain.Template{Name: name, Content: content}); err != nil {
		return err
	}
	if opts.openInEditor {
		editorAdapter := editor.New(cfg.Editor)
		if err := editorAdapter.Open(templatePath); err != nil {
			return err
		}
	}

	return printExitMessage(cmd.OutOrStdout(), cfg, fmt.Sprintf("Added template %s.", name), false)
}

func resolveAddInputs(args []string) (string, string, error) {
	name := ""
	content := ""
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		content = strings.Join(args[1:], " ")
	}
	if strings.TrimSpace(content) == "" {
		piped, err := readStdinIfPiped(os.Stdin)
		if err != nil {
			return "", "", err
		}
		if strings.TrimSpace(piped) != "" {
			content = piped
		}
	}
	return name, content, nil
}

type addTemplateStep int

const (
	addTemplateNameStep addTemplateStep = iota
	addTemplateContentStep
)

type addTemplateModel struct {
	step         addTemplateStep
	nameInput    textinput.Model
	contentInput textarea.Model
	canceled     bool
	errMessage   string
	theme        ui.Theme
}

func promptAddTemplate(defaultName, defaultContent string, theme ui.Theme) (string, string, bool, error) {
	model := newAddTemplateModel(defaultName, defaultContent, theme)
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return "", "", false, err
	}
	if m, ok := result.(addTemplateModel); ok {
		return strings.TrimSpace(m.nameInput.Value()), strings.TrimRight(m.contentInput.Value(), "\n"), m.canceled, nil
	}
	return "", "", false, fmt.Errorf("unexpected model result")
}

func newAddTemplateModel(defaultName, defaultContent string, theme ui.Theme) addTemplateModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "template-name"
	nameInput.CharLimit = 200
	nameInput.Width = 40
	nameInput.SetValue(strings.TrimSpace(defaultName))
	nameInput.Focus()

	contentInput := textarea.New()
	contentInput.Placeholder = "Template content"
	contentInput.SetValue(defaultContent)
	contentInput.CharLimit = 8000
	contentInput.SetHeight(8)
	contentInput.SetWidth(80)
	contentInput.Blur()

	return addTemplateModel{
		step:         addTemplateNameStep,
		nameInput:    nameInput,
		contentInput: contentInput,
		theme:        theme,
	}
}

func (m addTemplateModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m addTemplateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.step == addTemplateNameStep {
				if strings.TrimSpace(m.nameInput.Value()) == "" {
					m.errMessage = "Template name is required."
					return m, nil
				}
				m.step = addTemplateContentStep
				m.nameInput.Blur()
				m.contentInput.Focus()
				m.errMessage = ""
				return m, nil
			}
		case tea.KeyCtrlS:
			if m.step == addTemplateContentStep {
				if strings.TrimSpace(m.contentInput.Value()) == "" {
					m.errMessage = "Template content is required."
					return m, nil
				}
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	if m.step == addTemplateNameStep {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		m.contentInput, cmd = m.contentInput.Update(msg)
	}
	return m, cmd
}

func (m addTemplateModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Headings).Render("Add template")
	help := "Enter name, then press Enter. Add content, then press Ctrl+S to save."
	if m.errMessage != "" {
		help = lipgloss.NewStyle().Foreground(m.theme.TextHighlight).Render(m.errMessage)
	} else {
		help = lipgloss.NewStyle().Foreground(m.theme.Muted).Render(help)
	}

	nameBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(0, 1).
		Render(m.nameInput.View())
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(0, 1).
		Render(m.contentInput.View())

	return strings.Join([]string{
		title,
		help,
		"",
		"Name:",
		nameBox,
		"",
		"Content:",
		contentBox,
	}, "\n")
}
