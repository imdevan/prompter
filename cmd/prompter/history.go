package main

import (
	"fmt"
	"io"
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

	"prompter-cli/internal/adapters/clipboard"
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
	cmd.Flags().BoolVarP(&opts.insert, "insert", "n", false, "open history entry for insertion")
	return cmd
}

type historyOptions struct {
	clear  bool
	yes    bool
	keep   bool
	open   bool
	insert bool
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
	editorAdapter := editor.New(cfg.Editor)
	clip := clipboard.Adapter{}
	out := cmd.OutOrStdout()
	theme := ui.ThemeFromConfig(cfg)
	if opts.open {
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
			return openHistoryForInsert(out, theme, editorAdapter, &clip, entries[index-1].Path, opts.insert, cfg.PromptSeparator, cfg.Target)
		}
		tag := strings.TrimSpace(args[0])
		if tag != "" {
			return runHistoryTagSearch(cmd, cfg, entries, tag, opts.insert)
		}
		return err
	}

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
	path := strings.TrimSpace(m.selectedPath)
	if strings.TrimSpace(m.insertPath) != "" {
		path = m.insertPath
		opts.insert = true
	}
	if path == "" {
		return nil
	}
	return openHistoryForInsert(out, theme, editorAdapter, &clip, path, opts.insert, cfg.PromptSeparator, cfg.Target)
}

type historyEntry struct {
	Name       string
	Path       string
	ModTime    time.Time
	Size       int64
	Tag        string
	Flags      string
	BodyBytes  int
	BodyTokens int
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
		bodyBytes := 0
		bodyTokens := 0
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		body := strings.TrimSpace(template.StripFrontmatter(string(data)))
		if body != "" {
			bodyBytes = len([]byte(body))
			bodyTokens = len(body) / 4
		}
		items = append(items, historyEntry{
			Name:       entry.Name(),
			Path:       path,
			ModTime:    info.ModTime(),
			Size:       info.Size(),
			Tag:        tag,
			Flags:      flags,
			BodyBytes:  bodyBytes,
			BodyTokens: bodyTokens,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ModTime.After(items[j].ModTime)
	})
	return items, nil
}

type historyListItem struct {
	entry       historyEntry
	tag         string
	timeLine    string
	description string
	filterValue string
}

func (h historyListItem) Title() string {
	return h.timeLine
}

func (h historyListItem) Description() string {
	return h.description
}

func (h historyListItem) FilterValue() string {
	return h.filterValue
}

type historyItemDelegate struct {
	list.DefaultDelegate
	theme ui.Theme
}

func (d historyItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(historyListItem)
	if !ok {
		return
	}
	titleStyle := d.Styles.NormalTitle
	descStyle := d.Styles.NormalDesc
	if index == m.Index() {
		titleStyle = d.Styles.SelectedTitle
		descStyle = d.Styles.SelectedDesc
	}
	title := titleStyle.Render(item.timeLine)
	if strings.TrimSpace(item.tag) != "" {
		tagStyle := titleStyle.Foreground(d.theme.Tags)
		title = tagStyle.Render("#"+item.tag) + "\n" + titleStyle.Render(item.timeLine)
	}
	desc := descStyle.Render(item.description)
	if d.ShowDescription {
		fmt.Fprintf(w, "%s\n%s", title, desc)
		return
	}
	fmt.Fprint(w, title)
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

type historyDisplay struct {
	tag         string
	timeLine    string
	description string
}

func formatHistoryDisplay(entry historyEntry, now time.Time, enableTimeAgo bool, dateTimeFormat string, flagWidth int, tokenWidth int) historyDisplay {
	const day = 24 * time.Hour
	const week = 7 * day
	const month = 30 * day

	tag := strings.TrimSpace(entry.Tag)
	age := now.Sub(entry.ModTime)
	if age < 0 {
		age = 0
	}

	timeLine := formatHistoryTimeLine(entry.ModTime, age, week, month, enableTimeAgo, dateTimeFormat)
	description := formatHistoryFileLine(entry.Flags, entry.BodyBytes, entry.BodyTokens, flagWidth, tokenWidth)
	return historyDisplay{
		tag:         tag,
		timeLine:    timeLine,
		description: description,
	}
}

func historyFilterValue(entry historyEntry, display historyDisplay) string {
	parts := []string{
		strings.TrimSpace(entry.Name),
		strings.TrimSpace(display.tag),
		strings.TrimSpace(entry.Flags),
	}
	return strings.TrimSpace(strings.Join(parts, " "))
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

func formatHistoryFileLine(flags string, bytes int, tokens int, flagWidth int, tokenWidth int) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(flags) != "" {
		parts = append(parts, " "+formatFixedWidth(strings.TrimSpace(flags), flagWidth, false))
	} else {
		parts = append(parts, " "+formatFixedWidth("_", flagWidth, false))
	}
	parts = append(parts, fmt.Sprintf(" %s", formatFixedWidth(fmt.Sprintf("~%d", tokens), tokenWidth, false)))
	parts = append(parts, formatSize(int64(bytes)))
	return strings.Join(parts, " • ")
}

const (
	maxHistoryFlagWidthLimit  = 5
	maxHistoryTokenWidthLimit = 5
)

func maxHistoryFlagWidth(entries []historyEntry) int {
	width := 1
	for _, entry := range entries {
		if strings.TrimSpace(entry.Flags) == "" {
			continue
		}
		count := len([]rune(entry.Flags))
		if count > width {
			width = count
		}
		if width >= maxHistoryFlagWidthLimit {
			return maxHistoryFlagWidthLimit
		}
	}
	if width > maxHistoryFlagWidthLimit {
		return maxHistoryFlagWidthLimit
	}
	return width
}

func maxHistoryTokenWidth(entries []historyEntry) int {
	width := 1
	for _, entry := range entries {
		value := fmt.Sprintf("~%d", entry.BodyTokens)
		count := len([]rune(value))
		if count > width {
			width = count
		}
		if width >= maxHistoryTokenWidthLimit {
			return maxHistoryTokenWidthLimit
		}
	}
	if width > maxHistoryTokenWidthLimit {
		return maxHistoryTokenWidthLimit
	}
	return width
}

func formatFixedWidth(value string, width int, padLeft bool) string {
	runes := []rune(value)
	if len(runes) > width {
		return value
	}
	value = string(runes)
	if padLeft {
		return fmt.Sprintf("%*s", width, value)
	}
	return fmt.Sprintf("%-*s", width, value)
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

func openHistoryForInsert(out io.Writer, theme ui.Theme, editorAdapter *editor.Adapter, clip *clipboard.Adapter, path string, insert bool, separator string, target string) error {
	if !insert {
		return editorAdapter.Open(path)
	}
	line, err := ensureHistoryInsertMarker(path, separator)
	if err != nil {
		return err
	}
	command := editor.ResolveCommand(editorAdapter.Command)
	if command == "" {
		return editorAdapter.Open(path)
	}
	if editor.IsVim(command) {
		if err := editor.OpenVimInsert(command, path, line); err != nil {
			return err
		}
		if strings.EqualFold(strings.TrimSpace(target), "clipboard") && clip != nil {
			if err := copyHistoryInsertToClipboard(clip, path, separator); err != nil {
				return err
			}
			return confirmClipboardCopy(out, theme)
		}
		return nil
	}
	return editorAdapter.Open(path)
}

func ensureHistoryInsertMarker(path string, separator string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	separator = normalizePromptSeparator(separator)
	content := string(data)
	header, body, hasFrontmatter := splitHistoryFrontmatter(content)
	body = strings.TrimLeft(body, "\r\n")
	insertLine := 2
	if hasFrontmatter {
		insertLine = countLines(header) + 2
	}
	if strings.HasPrefix(body, separator) {
		return insertLine, nil
	}
	insertBlock := "\n\n\n\n" + separator + "\n\n"
	if hasFrontmatter {
		content = header + insertBlock + body
	} else {
		content = insertBlock + body
	}
	return insertLine, os.WriteFile(path, []byte(content), 0o644)
}

func copyHistoryInsertToClipboard(clip *clipboard.Adapter, path string, separator string) error {
	if clip == nil {
		return fmt.Errorf("clipboard adapter is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	body := template.StripFrontmatter(string(data))
	body = strings.TrimLeft(body, "\r\n")
	separator = normalizePromptSeparator(separator)
	lines := strings.Split(body, "\n")
	firstSeparator := -1
	for i, line := range lines {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == separator {
			firstSeparator = i
			break
		}
	}
	if firstSeparator >= 0 {
		body = strings.Join(lines[:firstSeparator], "\n")
	}
	body = strings.TrimSpace(body)
	return clip.WriteText(body)
}

func splitHistoryFrontmatter(content string) (string, string, bool) {
	trimmed := strings.TrimLeft(content, "\ufeff\r\n\t ")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return "", content, false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", content, false
	}
	header := strings.Join(lines[:end+1], "\n")
	body := strings.Join(lines[end+1:], "\n")
	body = strings.TrimLeft(body, "\r\n")
	return header, body, true
}

func countLines(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}

func normalizePromptSeparator(separator string) string {
	trimmed := strings.TrimSpace(separator)
	if trimmed == "" {
		return "---"
	}
	return trimmed
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

func runHistoryTagSearch(cmd *cobra.Command, cfg domain.Config, entries []historyEntry, tag string, insert bool) error {
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
	clip := clipboard.Adapter{}
	out := cmd.OutOrStdout()
	theme := ui.ThemeFromConfig(cfg)
	if len(matches) == 1 {
		return openHistoryForInsert(out, theme, editorAdapter, &clip, matches[0].Path, insert, cfg.PromptSeparator, cfg.Target)
	}
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
	path := strings.TrimSpace(m.selectedPath)
	if strings.TrimSpace(m.insertPath) != "" {
		path = m.insertPath
		insert = true
	}
	if path == "" {
		return nil
	}
	return openHistoryForInsert(out, theme, editorAdapter, &clip, path, insert, cfg.PromptSeparator, cfg.Target)
}

func confirmClipboardCopy(out io.Writer, theme ui.Theme) error {
	if out == nil {
		return nil
	}
	_, err := fmt.Fprintln(out, ui.ClipboardConfirm(theme))
	return err
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
	insertPath    string
}

func newHistoryModel(entries []historyEntry, location string, enableTimeAgo bool, dateTimeFormat string, theme ui.Theme) historyModel {
	items := make([]list.Item, 0, len(entries))
	now := time.Now()
	flagWidth := maxHistoryFlagWidth(entries)
	tokenWidth := maxHistoryTokenWidth(entries)
	for _, entry := range entries {
		display := formatHistoryDisplay(entry, now, enableTimeAgo, dateTimeFormat, flagWidth, tokenWidth)
		items = append(items, historyListItem{
			entry:       entry,
			tag:         display.tag,
			timeLine:    display.timeLine,
			description: display.description,
			filterValue: historyFilterValue(entry, display),
		})
	}
	baseDelegate := ui.NewListDelegate(theme, ui.ListDelegateOptions{
		Height: 2,
	})
	delegate := historyItemDelegate{
		DefaultDelegate: baseDelegate,
		theme:           theme,
	}
	model := ui.NewListModel(items, delegate, 80, 20, theme)
	model.Title = "History"
	model.Styles.Title = lipgloss.NewStyle().Foreground(theme.Headings).Bold(true)
	model.SetShowStatusBar(false)
	model.SetShowHelp(false)
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
		if m.list.FilterState() == list.Filtering {
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
			break
		}
		if msg.Type == tea.KeyEsc && m.list.FilterState() == list.FilterApplied {
			break
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
		case "i", "insert":
			item, ok := m.list.SelectedItem().(historyListItem)
			if !ok || item.entry.Path == "" {
				return m, tea.Quit
			}
			m.insertPath = item.entry.Path
			return m, tea.Quit
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
		empty := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("No history entries found.")
		return empty
	}
	frame := ui.FrameStyle(m.theme)
	if !m.deleteMode {
		listView := m.list.View()
		helpView := ui.ListHelpView(m.list, m.shortHelpKeys(), m.fullHelpKeys())
		return frame.Render(listView + "\n" + helpView)
	}
	return frame.Render(m.deleteView())
}

func (m historyModel) shortHelpKeys() []key.Binding {
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
			key.WithKeys("i", "insert"),
			key.WithHelp("i/ins", "insert"),
		),
		m.list.KeyMap.ShowFullHelp,
	}
}

func (m historyModel) fullHelpKeys() [][]key.Binding {
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

func (m historyModel) applySize(width, height int) {
	ui.ApplyFrameListSize(&m.list, width, height, ui.FrameSizeOptions{})
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
	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Headings).Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	errorStyle := lipgloss.NewStyle().Foreground(m.theme.TextHighlight).Bold(true)
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
