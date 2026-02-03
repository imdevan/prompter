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
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/adapters/editor"
	"prompter-cli/internal/config"
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
	return cmd
}

type historyOptions struct {
	clear bool
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
	if opts.clear {
		if err := clearHistory(cfg.HistoryLocation); err != nil {
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
		if err != nil {
			return err
		}
		if index <= 0 || index > len(entries) {
			return fmt.Errorf("history index out of range (1-%d)", len(entries))
		}
		editorAdapter := editor.New(cfg.Editor)
		return editorAdapter.Open(entries[index-1].Path)
	}

	model := newHistoryModel(entries, cfg.HistoryLocation)
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
		items = append(items, historyEntry{
			Name:    entry.Name(),
			Path:    filepath.Join(dir, entry.Name()),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ModTime.After(items[j].ModTime)
	})
	return items, nil
}

type historyListItem struct {
	entry historyEntry
}

func (h historyListItem) Title() string {
	return h.entry.Name
}

func (h historyListItem) Description() string {
	timestamp := h.entry.ModTime.Local().Format("2006-01-02 15:04")
	size := formatSize(h.entry.Size)
	return fmt.Sprintf("%s • %s", timestamp, size)
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

func clearHistory(dir string) error {
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

type historyModel struct {
	list         list.Model
	location     string
	selectedPath string
}

func newHistoryModel(entries []historyEntry, location string) historyModel {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, historyListItem{entry: entry})
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("2")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("5"))
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("7")).Bold(true)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(lipgloss.Color("8"))
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
		BorderForeground(lipgloss.Color("#444"))
	return frame.Render(header + m.list.View() + "\n\n" + help)
}
