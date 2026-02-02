package bubbletea

import (
	"fmt"
	"io"
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
func (Adapter) SelectTemplates(templates []domain.Template, basePrompt string) ([]domain.Template, error) {
	model := newTemplateSelectModel(templates, basePrompt)
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
	list       list.Model
	templates  []domain.Template
	selecteds  map[int]bool
	order      []int
	basePrompt string
	barIndex   int
	focus      focusArea
}

type focusArea int

const (
	focusList focusArea = iota
	focusBar
)

func newTemplateSelectModel(templates []domain.Template, basePrompt string) templateSelectModel {
	items := make([]list.Item, 0, len(templates))
	for i, tmpl := range templates {
		items = append(items, templateItem{template: tmpl, index: i})
	}

	selecteds := make(map[int]bool)
	defaultDelegate := list.NewDefaultDelegate()
	defaultDelegate.Styles.SelectedTitle = defaultDelegate.Styles.SelectedTitle.Foreground(lipgloss.Color("2")).Bold(true)
	defaultDelegate.Styles.SelectedDesc = defaultDelegate.Styles.SelectedDesc.Foreground(lipgloss.Color("5"))
	delegate := templateItemDelegate{
		DefaultDelegate: defaultDelegate,
		selecteds:       selecteds,
	}

	l := list.New(items, delegate, 80, 20)
	l.Title = "Select templates"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)

	return templateSelectModel{
		list:       l,
		templates:  templates,
		selecteds:  selecteds,
		order:      make([]int, 0),
		basePrompt: strings.TrimSpace(basePrompt),
		barIndex:   0,
		focus:      focusList,
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
		case tea.KeyTab:
			if m.focus == focusList {
				m.focus = focusBar
			} else {
				m.focus = focusList
			}
			return m, nil
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeySpace:
			if m.focus == focusBar {
				m.toggleFocusedSelection()
				return m, nil
			}
			if item, ok := m.list.SelectedItem().(templateItem); ok {
				m.toggleSelection(item.index)
			}
		case tea.KeyLeft, tea.KeyRight:
			if m.focus == focusBar {
				m.moveBarIndex(msg.Type)
				return m, nil
			}
		case tea.KeyBackspace, tea.KeyDelete:
			if m.focus == focusBar {
				m.toggleFocusedSelection()
				return m, nil
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
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render("Space to toggle, Tab to focus summary, Enter to continue.")
	summary := m.renderSelectionBar()
	return lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444")).
		Render(header + "\n\n" + summary + "\n\n" + m.list.View())
}

func (m templateSelectModel) selected() []domain.Template {
	selected := make([]domain.Template, 0, len(m.order))
	for _, idx := range m.order {
		if m.selecteds[idx] {
			selected = append(selected, m.templates[idx])
		}
	}
	return selected
}

func (m *templateSelectModel) moveBarIndex(key tea.KeyType) {
	entries := m.selectionEntries()
	if len(entries) == 0 {
		m.barIndex = 0
		return
	}
	switch key {
	case tea.KeyLeft:
		m.barIndex--
	case tea.KeyRight:
		m.barIndex++
	}
	if m.barIndex < 0 {
		m.barIndex = 0
	}
	if m.barIndex >= len(entries) {
		m.barIndex = len(entries) - 1
	}
}

func (m *templateSelectModel) clampBarIndex() {
	entries := m.selectionEntries()
	if len(entries) == 0 {
		m.barIndex = 0
		return
	}
	if m.barIndex >= len(entries) {
		m.barIndex = len(entries) - 1
	}
}

func (m *templateSelectModel) toggleFocusedSelection() {
	entries := m.selectionEntries()
	if len(entries) == 0 || m.barIndex >= len(entries) {
		return
	}
	entry := entries[m.barIndex]
	if entry.templateIndex < 0 {
		return
	}
	m.toggleSelection(entry.templateIndex)
}

func (m *templateSelectModel) toggleSelection(index int) {
	if m.selecteds[index] {
		m.selecteds[index] = false
		m.removeFromOrder(index)
	} else {
		m.selecteds[index] = true
		m.order = append(m.order, index)
	}
	m.clampBarIndex()
}

func (m *templateSelectModel) removeFromOrder(index int) {
	for i, value := range m.order {
		if value == index {
			m.order = append(m.order[:i], m.order[i+1:]...)
			return
		}
	}
}

type selectionEntry struct {
	label         string
	templateIndex int
}

func (m templateSelectModel) selectionEntries() []selectionEntry {
	entries := make([]selectionEntry, 0, len(m.order)+1)
	for _, idx := range m.order {
		if !m.selecteds[idx] {
			continue
		}
		tmpl := m.templates[idx]
		label := tmpl.Name
		entries = append(entries, selectionEntry{
			label:         label,
			templateIndex: idx,
		})
	}
	if summary := summarizePrompt(m.basePrompt); summary != "" {
		entries = append(entries, selectionEntry{
			label:         summary,
			templateIndex: -1,
		})
	}
	return entries
}

func (m templateSelectModel) renderSelectionBar() string {
	entries := m.selectionEntries()
	if len(entries) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No templates selected yet.")
	}
	normal := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("7")).
		Foreground(lipgloss.Color("7"))
	focused := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("2")).
		Bold(true)
	parts := make([]string, 0, len(entries))
	for i, entry := range entries {
		chip := entry.label
		if m.focus == focusBar && i == m.barIndex {
			parts = append(parts, focused.Render(chip))
		} else {
			parts = append(parts, normal.Render(chip))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func summarizePrompt(prompt string) string {
	text := strings.Join(strings.Fields(prompt), " ")
	if text == "" {
		return ""
	}
	runes := []rune(text)
	const maxLen = 15
	if len(runes) <= maxLen {
		return fmt.Sprintf("%q", text)
	}
	return fmt.Sprintf("%q", string(runes[:maxLen])+"...")
}

type templateItemDelegate struct {
	list.DefaultDelegate
	selecteds map[int]bool
}

func (d templateItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(templateItem)
	if !ok {
		return
	}
	isSelected := d.selecteds[item.index]
	cursor := " "
	if index == m.Index() {
		cursor = ">"
	}
	marker := "[ ]"
	if isSelected {
		marker = "[x]"
	}
	titleStyle := d.Styles.NormalTitle
	descStyle := d.Styles.NormalDesc
	if index == m.Index() {
		titleStyle = d.Styles.SelectedTitle
		descStyle = d.Styles.SelectedDesc
	}

	title := titleStyle.Render(fmt.Sprintf("%s %s %s", cursor, marker, item.Title()))
	if desc := item.Description(); desc != "" {
		desc = descStyle.Render("    " + desc)
		fmt.Fprintf(w, "%s\n%s", title, desc)
		return
	}
	fmt.Fprint(w, title)
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
