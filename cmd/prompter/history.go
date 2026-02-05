package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/adapters/editor"
	"prompter-cli/internal/config"
	"prompter-cli/internal/domain"
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

	model := newHistoryModel(entries, cfg.HistoryLocation, cfg.HistoryEnableTimeAgo, cfg.HistoryDateTime)
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
	Name    string
	Path    string
	ModTime time.Time
	Size    int64
	Tag     string
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
		tag := extractHistoryTag(filepath.Join(dir, entry.Name()))
		items = append(items, historyEntry{
			Name:    entry.Name(),
			Path:    filepath.Join(dir, entry.Name()),
			ModTime: info.ModTime(),
			Size:    info.Size(),
			Tag:     tag,
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

func formatHistoryDisplay(entry historyEntry, now time.Time, enableTimeAgo bool, dateTimeFormat string) (string, string) {
	const day = 24 * time.Hour
	const week = 7 * day
	const month = 30 * day

	tag := strings.TrimSpace(entry.Tag)
	age := now.Sub(entry.ModTime)
	if age < 0 {
		age = 0
	}

	timeLine := formatHistoryTimeLine(entry.ModTime, age, week, month, enableTimeAgo, dateTimeFormat)
	fileLine := formatHistoryFileLine(entry.Name, entry.Size)

	if tag == "" {
		return timeLine, fileLine
	}

	tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	if age >= month || !enableTimeAgo {
		tagStyle = tagStyle.Bold(true)
	}
	title := tagStyle.Render("#" + tag)
	description := strings.Join([]string{timeLine, fileLine}, "\n")
	return title, description
}

func formatHistoryTimeLine(modTime time.Time, age time.Duration, week time.Duration, month time.Duration, enableTimeAgo bool, dateTimeFormat string) string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	localTime := modTime.Local()

	if !enableTimeAgo {
		return bold.Render(localTime.Format(historyDateTimeLayout(dateTimeFormat)))
	}

	switch {
	case age < 24*time.Hour:
		return bold.Render(formatTimeAgo(age))
	case age < month:
		if age < week {
			return bold.Render(formatTimeAgo(age))
		}
		return bold.Render(formatWeekdayOrdinalTime(localTime))
	default:
		return bold.Render(localTime.Format(historyDateTimeLayout(dateTimeFormat)))
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

func formatHistoryFileLine(name string, size int64) string {
	displayName := strings.TrimPrefix(name, "prompter-")
	displayName = strings.TrimSuffix(displayName, ".md")
	sizeText := formatSize(size)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(fmt.Sprintf("%s • %s", displayName, sizeText))
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
	model := newHistoryModel(matches, cfg.HistoryLocation, cfg.HistoryEnableTimeAgo, cfg.HistoryDateTime)
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
	list         list.Model
	location     string
	selectedPath string
}

func newHistoryModel(entries []historyEntry, location string, enableTimeAgo bool, dateTimeFormat string) historyModel {
	items := make([]list.Item, 0, len(entries))
	now := time.Now()
	for _, entry := range entries {
		title, description := formatHistoryDisplay(entry, now, enableTimeAgo, dateTimeFormat)
		items = append(items, historyListItem{
			entry:       entry,
			title:       title,
			description: description,
		})
	}
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(3)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Padding(0, 0, 0, 2)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().Padding(0, 0, 0, 2)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Padding(0, 0, 0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("2"))
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Padding(0, 0, 0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("2"))
	model := list.New(items, delegate, 80, 20)
	model.Title = "History"
	model.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	model.SetShowStatusBar(false)
	model.SetShowHelp(false)
	model.SetFilteringEnabled(true)
	return historyModel{
		list:     model,
		location: location,
	}
}

func (m historyModel) Init() tea.Cmd {
	return nil
}

func (m historyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
	case tea.WindowSizeMsg:
		width := msg.Width - 4
		height := msg.Height - 6
		if width < 40 {
			width = 40
		}
		if height < 8 {
			height = 8
		}
		m.list.SetSize(width, height)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m historyModel) View() string {
	if len(m.list.Items()) == 0 {
		empty := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render("No history entries found.")
		return empty
	}
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("Enter to open, Esc to exit.")
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	header := ""
	if strings.TrimSpace(m.location) != "" {
		header = pathStyle.Render(m.location) + "\n\n"
	}
	frame := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8"))
	return frame.Render(header + m.list.View() + "\n\n" + help)
}
