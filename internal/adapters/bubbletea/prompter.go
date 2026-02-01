package bubbletea

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"prompter-cli/internal/domain"
)

// Adapter implements interactive UI using Bubble Tea + Bubbles.
type Adapter struct{}

// AskBasePrompt prompts for the base prompt.
func (Adapter) AskBasePrompt(defaultValue, note string) (string, error) {
	model := newTextInputModel("Base prompt", "Enter your base prompt", defaultValue, note)
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return "", err
	}
	if m, ok := result.(textInputModel); ok {
		return strings.TrimSpace(m.input.Value()), nil
	}
	return "", fmt.Errorf("unexpected model result")
}

// SelectTemplates prompts for template selection.
func (Adapter) SelectTemplates(templates []domain.Template) ([]domain.Template, error) {
	model := newTemplateSelectModel(templates)
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return nil, err
	}
	if m, ok := result.(templateSelectModel); ok {
		return m.selected(), nil
	}
	return nil, fmt.Errorf("unexpected model result")
}

type textInputModel struct {
	title       string
	description string
	note        string
	input       textinput.Model
	ready       bool
}

func newTextInputModel(title, description, defaultValue, note string) textInputModel {
	input := textinput.New()
	input.Placeholder = description
	input.SetValue(defaultValue)
	input.Focus()
	input.CharLimit = 2000
	input.Width = 80

	return textInputModel{
		title:       title,
		description: description,
		note:        note,
		input:       input,
	}
}

func (m textInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m textInputModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render(m.title)
	description := lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render(m.description)
	body := lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#444")).Render(m.input.View())
	parts := []string{title, description}
	if strings.TrimSpace(m.note) != "" {
		note := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(m.note)
		parts = append(parts, note)
	}
	parts = append(parts, body, "Press Enter to continue.")
	return strings.Join(parts, "\n")
}

type templateSelectModel struct {
	list      list.Model
	templates []domain.Template
	selecteds map[int]bool
}

func newTemplateSelectModel(templates []domain.Template) templateSelectModel {
	items := make([]list.Item, 0, len(templates))
	for i, tmpl := range templates {
		items = append(items, templateItem{template: tmpl, index: i})
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("2")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("5"))

	l := list.New(items, delegate, 80, 20)
	l.Title = "Select templates"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)

	return templateSelectModel{
		list:      l,
		templates: templates,
		selecteds: make(map[int]bool),
	}
}

func (m templateSelectModel) Init() tea.Cmd {
	return nil
}

func (m templateSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeySpace:
			if item, ok := m.list.SelectedItem().(templateItem); ok {
				m.selecteds[item.index] = !m.selecteds[item.index]
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m templateSelectModel) View() string {
	if len(m.templates) == 0 {
		return "No templates available."
	}
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render("Space to toggle, Enter to continue.")
	return lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444")).
		Render(header + "\n\n" + m.list.View())
}

func (m templateSelectModel) selected() []domain.Template {
	selected := make([]domain.Template, 0, len(m.selecteds))
	for idx := range m.selecteds {
		if m.selecteds[idx] {
			selected = append(selected, m.templates[idx])
		}
	}
	return selected
}

type templateItem struct {
	template domain.Template
	index    int
}

func (t templateItem) Title() string {
	if t.template.Title != "" {
		return fmt.Sprintf("%s (%s)", t.template.Name, t.template.Title)
	}
	return t.template.Name
}

func (t templateItem) Description() string {
	return t.template.Description
}

func (t templateItem) FilterValue() string {
	return t.template.Name
}
