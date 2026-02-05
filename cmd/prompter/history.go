package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/adapters/editor"
	"prompter-cli/internal/config"
	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
	"prompter-cli/internal/ui"
)

func newHistoryCmd() *cobra.Command {
	opts := &historyOptions{}
	cmd := &cobra.Command{
		Use:   "history [index]",
		Short: "List saved prompt history",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistory(cmd, opts, args)
		},
	}
	cmd.Flags().BoolVarP(&opts.clear, "clear", "c", false, "clear prompt history")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "skip confirmation")
	cmd.Flags().BoolVarP(&opts.keep, "keep-tags", "k", false, "keep tagged history entries when clearing")
	cmd.Flags().BoolVarP(&opts.open, "editor", "e", false, "open history folder in editor")
	return cmd
}

type historyOptions struct {
	clear bool
	yes   bool
	keep  bool
	open  bool
}

func runHistory(cmd *cobra.Command, opts *historyOptions, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	manager := config.NewManager(cwd)
	cfg, err := manager.Load()
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.HistoryLocation) == "" {
		return fmt.Errorf("history_location is not configured")
	}
	if opts.open {
		editorAdapter := editor.New(cfg.Editor)
		return editorAdapter.Open(cfg.HistoryLocation)
	}
	if opts.clear {
		if !opts.yes {
			confirm, err := promptConfirmClear()
			if err != nil {
				return err
			}
			if !confirm {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "History not cleared.")
				return err
			}
		}
		if err := clearHistory(cfg.HistoryLocation, opts.keep); err != nil {
			return err
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "History cleared.")
		return err
	}

	if err := pruneHistoryEntries(cfg, cwd); err != nil {
		return err
	}
	entries, err := readHistoryEntries(cfg.HistoryLocation)
	if err != nil {
		return err
	}
	if len(args) == 1 {
		index, err := parseHistoryIndex(args[0])
		if err == nil {
			if index <= 0 || index > len(entries) {
				return fmt.Errorf("history index out of range (1-%d)", len(entries))
			}
			editorAdapter := editor.New(cfg.Editor)
			return editorAdapter.Open(entries[index-1].Path)
		}
		tag := strings.TrimSpace(args[0])
		if tag != "" {
			return runHistoryTagSearch(cmd, cfg, entries, tag)
		}
		return err
	}

	theme := ui.ThemeFromConfig(cfg)
	model := newHistoryModel(entries, cfg.HistoryLocation, cfg.HistoryEnableTimeAgo, cfg.HistoryDateTime, theme)
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return err
	}
	m, ok := result.(historyModel)
	if !ok {
		return fmt.Errorf("unexpected history model result")
	}
	if strings.TrimSpace(m.selectedPath) == "" {
		return nil
	}
	editorAdapter := editor.New(cfg.Editor)
	return editorAdapter.Open(m.selectedPath)
}

type historyEntry struct {
	Name      string
	Path      string
	ModTime   time.Time
	Size      int64
	Tag       string
	Flags     string
	BodyLines int
	BodyBytes int
}

func readHistoryEntries(dir string) ([]historyEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	items := make([]historyEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		path := filepath.Join(dir, entry.Name())
		tag := extractHistoryTag(path)
		flags := historyFlagsFromName(entry.Name())
		bodyLines := 0
		bodyBytes := 0
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		body := strings.TrimSpace(template.StripFrontmatter(string(data)))
		if body != "" {
			bodyLines = strings.Count(body, "\n") + 1
			bodyBytes = len([]byte(body))
		}
		items = append(items, historyEntry{
			Name:      entry.Name(),
			Path:      path,
			ModTime:   info.ModTime(),
			Size:      info.Size(),
			Tag:       tag,
			Flags:     flags,
			BodyLines: bodyLines,
			BodyBytes: bodyBytes,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ModTime.After(items[j].ModTime)
	})
	return items, nil
}

type historyListItem struct {
	entry       historyEntry
	title       string
	description string
}

func (h historyListItem) Title() string {
	return h.title
}

func (h historyListItem) Description() string {
	return h.description
}

func (h historyListItem) FilterValue() string {
	return h.entry.Name
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB"}
	value := float64(size)
	for _, unit := range units {
		value = value / 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f TB", value/1024)
}

func historyFlagsFromName(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.TrimPrefix(base, "prompt-")
	parts := strings.Split(base, "-")
	if len(parts) <= 2 {
		return ""
	}
	return strings.Join(parts[2:], "-")
}

func formatHistoryDisplay(entry historyEntry, now time.Time, enableTimeAgo bool, dateTimeFormat string, theme ui.Theme) (string, string) {
	const day = 24 * time.Hour
	const week = 7 * day
	const month = 30 * day

	tag := strings.TrimSpace(entry.Tag)
	age := now.Sub(entry.ModTime)
	if age < 0 {
		age = 0
	}

	timeLine := formatHistoryTimeLine(entry.ModTime, age, week, month, enableTimeAgo, dateTimeFormat)
	title := timeLine
	if tag != "" {
		tagStyle := lipgloss.NewStyle().Foreground(theme.Accent)
		title = tagStyle.Render("#"+tag) + "\n" + timeLine
	}
	description := formatHistoryFileLine(entry.Flags, entry.BodyLines, entry.BodyBytes)
	return title, description
}

func formatHistoryTimeLine(modTime time.Time, age time.Duration, week time.Duration, month time.Duration, enableTimeAgo bool, dateTimeFormat string) string {
	localTime := modTime.Local()

	if !enableTimeAgo {
		return localTime.Format(historyDateTimeLayout(dateTimeFormat))
	}

	switch {
	case age < 24*time.Hour:
		return formatTimeAgo(age)
	case age < month:
		if age < week {
			return formatTimeAgo(age)
		}
		return formatWeekdayOrdinalTime(localTime)
	default:
		return localTime.Format(historyDateTimeLayout(dateTimeFormat))
	}
}

func formatTimeAgo(age time.Duration) string {
	if age < time.Minute {
		return "Just now"
	}
	if age < 2*time.Minute {
		return "One minute ago"
	}
	if age < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(age.Minutes()))
	}
	if age < 2*time.Hour {
		return "One hour ago"
	}
	if age < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(age.Hours()))
	}
	if age < 48*time.Hour {
		return "One day ago"
	}
	return fmt.Sprintf("%d days ago", int(age.Hours()/24))
}

func formatWeekdayOrdinalTime(t time.Time) string {
	weekday := map[time.Weekday]string{
		time.Monday:    "Mon.",
		time.Tuesday:   "Tues.",
		time.Wednesday: "Wed.",
		time.Thursday:  "Thur.",
		time.Friday:    "Fri.",
		time.Saturday:  "Sat.",
		time.Sunday:    "Sun.",
	}[t.Weekday()]
	day := t.Day()
	return fmt.Sprintf("%s %d%s, %s", weekday, day, ordinalSuffix(day), t.Format("15:04"))
}

func ordinalSuffix(day int) string {
	if day%100 >= 11 && day%100 <= 13 {
		return "th"
	}
	switch day % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

func formatHistoryFileLine(flags string, lines int, bytes int) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(flags) != "" {
		parts = append(parts, " "+strings.TrimSpace(flags))
	} else {
		parts = append(parts, " _")
	}
	parts = append(parts, fmt.Sprintf("󰦪 %d", lines))
	parts = append(parts, formatSize(int64(bytes)))
	return strings.Join(parts, " • ")
}

func historyDateTimeLayout(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "day, month", "day-month", "day month", "day_month":
		return "2 Jan 2006 15:04"
	case "month, day", "month-day", "month day", "month_day":
		return "Jan 2 2006 15:04"
	case "iso", "iso8601":
		return "2006-01-02 15:04"
	default:
		return value
	}
}

func clearHistory(dir string, keepTags bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if keepTags {
			tag := extractHistoryTag(filepath.Join(dir, entry.Name()))
			if strings.TrimSpace(tag) != "" {
				continue
			}
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func pruneHistoryEntries(cfg domain.Config, cwd string) error {
	if strings.TrimSpace(cfg.HistoryLocation) == "" {
		return nil
	}
	localPrompts := ""
	if strings.TrimSpace(cfg.LocalPromptsLocation) != "" {
		localPrompts = filepath.Join(cwd, cfg.LocalPromptsLocation)
	}
	indexBody := ""
	repo := template.NewRepository(localPrompts, cfg.PromptsLocation)
	if tmpl, err := repo.Get("index"); err == nil {
		indexBody = strings.TrimSpace(template.StripFrontmatter(tmpl.Content))
	}
	entries, err := os.ReadDir(cfg.HistoryLocation)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(cfg.HistoryLocation, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := strings.TrimSpace(template.StripFrontmatter(string(data)))
		if body == "" || (indexBody != "" && body == indexBody) {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseHistoryIndex(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("history index is required")
	}
	index, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid history index %q", value)
	}
	return index, nil
}

func promptConfirmClear() (bool, error) {
	confirmed := false
	err := huh.NewConfirm().
		Title("Clear history?").
		Description("Delete all saved prompts?").
		Value(&confirmed).
		Run()
	return confirmed, err
}

func extractHistoryTag(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.TrimSpace(strings.ToLower(key)) != "tag" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")
		return value
	}
	return ""
}

func runHistoryTagSearch(cmd *cobra.Command, cfg domain.Config, entries []historyEntry, tag string) error {
	matches := make([]historyEntry, 0)
	for _, entry := range entries {
		entryTag := strings.ToLower(strings.TrimSpace(entry.Tag))
		query := strings.ToLower(strings.TrimSpace(tag))
		if entryTag != "" && query != "" && strings.Contains(entryTag, query) {
			matches = append(matches, entry)
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("no history entries found with tag %q", tag)
	}
	editorAdapter := editor.New(cfg.Editor)
	if len(matches) == 1 {
		return editorAdapter.Open(matches[0].Path)
	}
	theme := ui.ThemeFromConfig(cfg)
	model := newHistoryModel(matches, cfg.HistoryLocation, cfg.HistoryEnableTimeAgo, cfg.HistoryDateTime, theme)
	program := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return err
	}
	m, ok := result.(historyModel)
	if !ok {
		return fmt.Errorf("unexpected history model result")
	}
	if strings.TrimSpace(m.selectedPath) == "" {
		return nil
	}
	return editorAdapter.Open(m.selectedPath)
}

type historyModel struct {
	list          list.Model
	location      string
	selectedPath  string
	theme         ui.Theme
	deleteMode    bool
	deleteIndex   int
	deleteCount   int
	deleteContent string
	deleteItem    historyListItem
	deleteError   string
}

func newHistoryModel(entries []historyEntry, location string, enableTimeAgo bool, dateTimeFormat string, theme ui.Theme) historyModel {
	items := make([]list.Item, 0, len(entries))
	now := time.Now()
	for _, entry := range entries {
		title, description := formatHistoryDisplay(entry, now, enableTimeAgo, dateTimeFormat, theme)
		items = append(items, historyListItem{
			entry:       entry,
			title:       title,
			description: description,
		})
	}
	delegate := ui.NewListDelegate(theme, ui.ListDelegateOptions{
		Height: 2,
	})
	delegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("󰌑"),
				key.WithHelp("󰌑", "Open"),
			),
			key.NewBinding(
				key.WithKeys("d", "D"),
				key.WithHelp("d/D", "delete"),
			),
			key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "Exit"),
			),
		}
	}
	model := list.New(items, delegate, 80, 20)
	model.Title = "History"
	model.Styles.Title = lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
	model.SetShowStatusBar(false)
	model.SetShowHelp(true)
	model.SetFilteringEnabled(true)
	return historyModel{
		list:     model,
		location: location,
		theme:    theme,
	}
}

func (m historyModel) Init() tea.Cmd {
	return nil
}

func (m historyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.deleteMode {
			return m.updateDelete(msg)
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			item, ok := m.list.SelectedItem().(historyListItem)
			if !ok || item.entry.Path == "" {
				return m, tea.Quit
			}
			m.selectedPath = item.entry.Path
			return m, tea.Quit
		}
		switch msg.String() {
		case "d", "backspace":
			return m.startDeleteConfirm()
		case "D", "delete":
			return m.deleteSelected()
		}
	case tea.WindowSizeMsg:
		m.applySize(msg.Width, msg.Height)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m historyModel) View() string {
	if len(m.list.Items()) == 0 {
		empty := lipgloss.NewStyle().Foreground(m.theme.BasePrompt).Render("No history entries found.")
		return empty
	}
	pathStyle := lipgloss.NewStyle().Foreground(m.theme.Secondary)
	header := ""
	if strings.TrimSpace(m.location) != "" {
		header = pathStyle.Render(m.location) + "\n\n"
	}
	frame := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border)
	if !m.deleteMode {
		return frame.Render(header + m.list.View())
	}
	return frame.Render(m.deleteView())
}

func (m historyModel) applySize(width, height int) {
	listWidth := width - 4
	listHeight := height - 6
	if listWidth < 40 {
		listWidth = 40
	}
	if listHeight < 8 {
		listHeight = 8
	}
	m.list.SetSize(listWidth, listHeight)
}

func (m historyModel) startDeleteConfirm() (tea.Model, tea.Cmd) {
	item, ok := m.list.SelectedItem().(historyListItem)
	if !ok || item.entry.Path == "" {
		return m, nil
	}
	content, err := loadHistoryPrompt(item.entry.Path)
	m.deleteMode = true
	m.deleteItem = item
	m.deleteIndex = m.list.Index()
	m.deleteCount = len(m.list.VisibleItems())
	m.deleteContent = content
	m.deleteError = ""
	if err != nil {
		m.deleteError = err.Error()
	}
	return m, nil
}

func (m historyModel) deleteSelected() (tea.Model, tea.Cmd) {
	item, ok := m.list.SelectedItem().(historyListItem)
	if !ok || item.entry.Path == "" {
		return m, nil
	}
	return m.deleteItemPath(item, m.list.Index(), len(m.list.VisibleItems()))
}

func (m historyModel) deleteItemPath(item historyListItem, visibleIndex int, visibleCount int) (tea.Model, tea.Cmd) {
	if err := os.Remove(item.entry.Path); err != nil {
		m.deleteError = err.Error()
		m.deleteMode = true
		m.deleteItem = item
		m.deleteIndex = visibleIndex
		m.deleteCount = visibleCount
		return m, nil
	}
	m.deleteMode = false
	m.deleteError = ""
	m.deleteContent = ""
	items := m.list.Items()
	nextItems := make([]list.Item, 0, len(items))
	for _, listItem := range items {
		historyItem, ok := listItem.(historyListItem)
		if ok && historyItem.entry.Path == item.entry.Path {
			continue
		}
		nextItems = append(nextItems, listItem)
	}
	cmd := m.list.SetItems(nextItems)
	if len(nextItems) == 0 {
		return m, cmd
	}
	newVisibleCount := visibleCount - 1
	if newVisibleCount <= 0 {
		return m, cmd
	}
	nextIndex := visibleIndex
	if nextIndex >= newVisibleCount {
		nextIndex = newVisibleCount - 1
	}
	if nextIndex < 0 {
		nextIndex = 0
	}
	m.list.Select(nextIndex)
	return m, cmd
}

func (m historyModel) updateDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		return m.deleteItemPath(m.deleteItem, m.deleteIndex, m.deleteCount)
	case "n", "N", "esc", "ctrl+c":
		m.deleteMode = false
		m.deleteError = ""
		return m, nil
	}
	return m, nil
}

func (m historyModel) deleteView() string {
	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(m.theme.Secondary)
	errorStyle := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	name := strings.TrimSuffix(m.deleteItem.entry.Name, filepath.Ext(m.deleteItem.entry.Name))
	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.Border).
		Padding(0, 1).
		MaxHeight(12)
	lines := []string{
		titleStyle.Render("Delete history item?"),
		promptStyle.Render(name),
		"",
		contentStyle.Render(m.deleteContent),
		"",
		promptStyle.Render("y/enter delete • n/esc cancel"),
	}
	if strings.TrimSpace(m.deleteError) != "" {
		lines = append(lines, errorStyle.Render("Error: "+m.deleteError))
	}
	return strings.Join(lines, "\n")
}

func loadHistoryPrompt(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "Unable to load history entry.", err
	}
	return strings.TrimLeft(string(data), "\n"), nil
}
