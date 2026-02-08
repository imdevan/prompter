package bubbletea

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/interactive"
	"prompter-cli/internal/ui"
)

// Adapter implements interactive UI using Bubble Tea + Bubbles.
type Adapter struct {
	Theme          ui.Theme
	AltEnterSubmit bool
}

// NewAdapter returns a Bubble Tea adapter configured with theme colors.
func NewAdapter(cfg domain.Config) Adapter {
	return Adapter{
		Theme:          ui.ThemeFromConfig(cfg),
		AltEnterSubmit: cfg.AltEnterSubmit,
	}
}

// AskBasePrompt prompts for the base prompt.
func (a Adapter) AskBasePrompt(defaultValue, note string) (string, error) {
	model := newTextInputModel("Base prompt", "Enter your base prompt", defaultValue, note, a.Theme, a.AltEnterSubmit)
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return "", err
	}
	if m, ok := result.(textInputModel); ok {
		if m.canceled {
			return "", interactive.ErrCanceled
		}
		return strings.TrimSpace(m.input.Value()), nil
	}
	return "", fmt.Errorf("unexpected model result")
}

// SelectTemplates prompts for template selection.
func (a Adapter) SelectTemplates(templates []domain.Template, basePrompt string, preselected []string) ([]domain.Template, error) {
	model := newTemplateSelectModel(templates, basePrompt, preselected, a.Theme)
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return nil, err
	}
	if m, ok := result.(templateSelectModel); ok {
		if m.canceled {
			return nil, interactive.ErrCanceled
		}
		return m.selected(), nil
	}
	return nil, fmt.Errorf("unexpected model result")
}

type textInputModel struct {
	title       string
	description string
	note        string
	input       textarea.Model
	ready       bool
	theme       ui.Theme
	canceled    bool
	submitKey   key.Binding
	newlineKey  key.Binding
}

func newTextInputModel(title, description, defaultValue, note string, theme ui.Theme, altEnterSubmit bool) textInputModel {
	input := textarea.New()
	input.Placeholder = description
	input.SetValue(strings.TrimSpace(defaultValue))
	input.Focus()
	input.CharLimit = 2000
	input.SetWidth(80)
	input.SetHeight(3)
	input.ShowLineNumbers = false
	input.FocusedStyle.Base = lipgloss.NewStyle().Foreground(theme.Text)
	input.BlurredStyle.Base = lipgloss.NewStyle().Foreground(theme.Text)
	submitKey := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit"))
	newlineKey := key.NewBinding(
		key.WithKeys("alt+enter", "shift+enter", "ctrl+j"),
		key.WithHelp("alt+enter", "newline"),
	)
	if altEnterSubmit {
		submitKey = key.NewBinding(key.WithKeys("alt+enter"), key.WithHelp("alt+enter", "submit"))
		newlineKey = key.NewBinding(
			key.WithKeys("enter", "shift+enter", "ctrl+j"),
			key.WithHelp("enter", "newline"),
		)
	}
	input.KeyMap.InsertNewline = newlineKey

	return textInputModel{
		title:       title,
		description: description,
		note:        note,
		input:       input,
		theme:       theme,
		submitKey:   submitKey,
		newlineKey:  newlineKey,
	}
}

func (m textInputModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.submitKey):
			return m, tea.Quit
		case msg.Type == tea.KeyCtrlC, msg.Type == tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.applySize(msg.Width, msg.Height)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m textInputModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Headings).Render(m.title)
	description := lipgloss.NewStyle().Foreground(m.theme.Text).Render(m.description)
	body := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(m.theme.Border).Render(m.input.View())
	parts := []string{title, description}
	if strings.TrimSpace(m.note) != "" {
		note := lipgloss.NewStyle().Foreground(m.theme.Muted).Render(m.note)
		parts = append(parts, note)
	}
	help := "Press Enter to continue. Alt+Enter (or Shift+Enter/Ctrl+J) for a new line.\n"
	if m.submitKey.Help().Key == "alt+enter" {
		help = "Press Alt+Enter to continue. Enter (or Shift+Enter/Ctrl+J) for a new line."
	}
	parts = append(parts, body, help)
	return lipgloss.NewStyle().Margin(1, 1).Render(strings.Join(parts, "\n"))
}

func (m *textInputModel) applySize(width, height int) {
	contentWidth := width - 6
	if contentWidth < 40 {
		contentWidth = 40
	}
	m.input.SetWidth(contentWidth)
	contentHeight := height - 8
	if contentHeight < 3 {
		contentHeight = 3
	}
	if contentHeight > 3 {
		contentHeight = 3
	}
	m.input.SetHeight(contentHeight)
}

type templateSelectModel struct {
	list       list.Model
	templates  []domain.Template
	selecteds  map[int]bool
	order      []int
	basePrompt string
	// todo: summary manipulation deferred to later version
	// barIndex   int
	// focus      focusArea
	theme    ui.Theme
	canceled bool
}

// todo: summary manipulation deferred to later version
// type focusArea int
//
// const (
// 	focusList focusArea = iota
// 	focusBar
// )

func newTemplateSelectModel(templates []domain.Template, basePrompt string, preselected []string, theme ui.Theme) templateSelectModel {
	items := make([]list.Item, 0, len(templates))
	for i, tmpl := range templates {
		items = append(items, templateItem{template: tmpl, index: i})
	}

	selecteds := make(map[int]bool)
	order := make([]int, 0, len(preselected))
	indexByName := make(map[string]int, len(templates))
	for i, tmpl := range templates {
		indexByName[strings.ToLower(tmpl.Name)] = i
	}
	for _, name := range preselected {
		idx, ok := indexByName[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			continue
		}
		if selecteds[idx] {
			continue
		}
		selecteds[idx] = true
		order = append(order, idx)
	}
	defaultDelegate := ui.NewListDelegate(theme, ui.ListDelegateOptions{})
	delegate := templateItemDelegate{
		DefaultDelegate: defaultDelegate,
		selecteds:       selecteds,
	}

	l := ui.NewListModel(items, delegate, 80, 20, theme)
	l.Title = "Select templates"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().Foreground(theme.Headings).Bold(true)

	return templateSelectModel{
		list:       l,
		templates:  templates,
		selecteds:  selecteds,
		order:      order,
		basePrompt: strings.TrimSpace(basePrompt),
		theme:      theme,
	}
}

func (m templateSelectModel) Init() tea.Cmd {
	return nil
}

func (m templateSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			if msg.Type == tea.KeyCtrlC {
				m.canceled = true
				return m, tea.Quit
			}
			break
		}
		if msg.Type == tea.KeyEsc && m.list.FilterState() == list.FilterApplied {
			break
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit
		// todo: summary manipulation deferred to later version
		// case tea.KeyTab:
		// 	if m.focus == focusList {
		// 		m.focus = focusBar
		// 	} else {
		// 		m.focus = focusList
		// 	}
		// 	return m, nil
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeySpace:
			// todo: summary manipulation deferred to later version
			// if m.focus == focusBar {
			// 	m.toggleFocusedSelection()
			// 	return m, nil
			// }
			if item, ok := m.list.SelectedItem().(templateItem); ok {
				m.toggleSelection(item.index)
			}
			// todo: summary manipulation deferred to later version
			// case tea.KeyLeft, tea.KeyRight:
			// 	if m.focus == focusBar {
			// 		m.moveBarIndex(msg.Type)
			// 		return m, nil
			// 	}
			// case tea.KeyBackspace, tea.KeyDelete:
			// 	if m.focus == focusBar {
			// 		m.toggleFocusedSelection()
			// 		return m, nil
			// 	}
		}
	case tea.WindowSizeMsg:
		m.applySize(msg.Width, msg.Height)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m templateSelectModel) View() string {
	if len(m.templates) == 0 {
		return "No templates available."
	}
	summary := m.renderSelectionBar()
	listView := m.list.View()
	helpView := ui.ListHelpView(m.list, m.shortHelpKeys(), m.fullHelpKeys())
	return ui.FrameStyle(m.theme).Render(summary + "\n\n" + listView + "\n" + helpView)
}

func (m templateSelectModel) shortHelpKeys() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("󱁐"),
			key.WithHelp("󱁐", "Toggle"),
		),
		key.NewBinding(
			key.WithKeys("󰌑"),
			key.WithHelp("󰌑", "Continue"),
		),
		m.list.KeyMap.Filter,
		m.list.KeyMap.ShowFullHelp,
	}
}

func (m templateSelectModel) fullHelpKeys() [][]key.Binding {
	short := m.shortHelpKeys()
	sections := [][]key.Binding{
		{
			short[0],
			short[1],
			short[2],
			short[3],
		},
	}
	sections = append(sections, ui.ListFullHelpSections(m.list, ui.ListHelpOptions{
		IncludeFilter: true,
		IncludePaging: true,
		IncludeQuit:   true,
	})...)
	return sections
}

func (m *templateSelectModel) applySize(width, height int) {
	ui.ApplyFrameListSize(&m.list, width, height, ui.FrameSizeOptions{
		VerticalInset: 12,
	})
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

// todo: summary manipulation deferred to later version
// func (m *templateSelectModel) moveBarIndex(key tea.KeyType) {
// 	entries := m.selectionEntries()
// 	if len(entries) == 0 {
// 		m.barIndex = 0
// 		return
// 	}
// 	switch key {
// 	case tea.KeyLeft:
// 		m.barIndex--
// 	case tea.KeyRight:
// 		m.barIndex++
// 	}
// 	if m.barIndex < 0 {
// 		m.barIndex = 0
// 	}
// 	if m.barIndex >= len(entries) {
// 		m.barIndex = len(entries) - 1
// 	}
// }
//
// func (m *templateSelectModel) clampBarIndex() {
// 	entries := m.selectionEntries()
// 	if len(entries) == 0 {
// 		m.barIndex = 0
// 		return
// 	}
// 	if m.barIndex >= len(entries) {
// 		m.barIndex = len(entries) - 1
// 	}
// }
//
// func (m *templateSelectModel) toggleFocusedSelection() {
// 	entries := m.selectionEntries()
// 	if len(entries) == 0 || m.barIndex >= len(entries) {
// 		return
// 	}
// 	entry := entries[m.barIndex]
// 	if entry.templateIndex < 0 {
// 		return
// 	}
// 	m.toggleSelection(entry.templateIndex)
// }

func (m *templateSelectModel) toggleSelection(index int) {
	if m.selecteds[index] {
		m.selecteds[index] = false
		m.removeFromOrder(index)
	} else {
		m.selecteds[index] = true
		m.order = append(m.order, index)
	}
	// todo: summary manipulation deferred to later version
	// m.clampBarIndex()
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
		emptyMessage := lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Muted).
			Foreground(m.theme.Muted).
			Render("empty")

		// TODO: deciding between "empty and \n\n"
		// return "\n\n"
		return emptyMessage
	}
	normal := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Foreground(m.theme.Text)
	// todo: summary manipulation deferred to later version
	// focused := lipgloss.NewStyle().
	// 	Padding(0, 1).
	// 	Border(lipgloss.RoundedBorder()).
	// 	BorderForeground(m.theme.Primary).
	// 	Foreground(m.theme.Text).
	// 	Background(m.theme.Primary).
	// 	Bold(true)
	basePromptStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BasePromptBadge).
		Foreground(m.theme.BasePromptBadge)
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		chip := entry.label
		if entry.templateIndex < 0 {
			parts = append(parts, basePromptStyle.Render(chip))
			continue
		}
		parts = append(parts, normal.Render(chip))
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
	name := strings.TrimSpace(t.template.Name)
	if name == "" {
		return strings.TrimSpace(t.template.Title)
	}
	return name
}

func (t templateItem) Description() string {
	return t.template.Description
}

func (t templateItem) FilterValue() string {
	return t.template.Name
}
