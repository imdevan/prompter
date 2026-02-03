package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/config"
)

func newHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "List saved prompt history",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistory(cmd)
		},
	}
}

func runHistory(cmd *cobra.Command) error {
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
	entries, err := readHistoryEntries(cfg.HistoryLocation)
	if err != nil {
		return err
	}
	content := renderHistory(entries, cfg.HistoryLocation)
	_, err = fmt.Fprintln(cmd.OutOrStdout(), content)
	return err
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

func renderHistory(entries []historyEntry, location string) string {
	var builder strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	builder.WriteString(titleStyle.Render("History"))
	if strings.TrimSpace(location) != "" {
		builder.WriteString("\n")
		builder.WriteString(pathStyle.Render(location))
	}
	builder.WriteString("\n\n")

	if len(entries) == 0 {
		builder.WriteString(descStyle.Render("No history entries found."))
		return builder.String()
	}

	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, historyListItem{entry: entry})
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.NormalTitle
	delegate.Styles.SelectedDesc = delegate.Styles.NormalDesc
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("5")).Bold(true)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(lipgloss.Color("7"))
	model := list.New(items, delegate, 80, len(items)+2)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.SetShowPagination(false)
	model.DisableQuitKeybindings()
	model.SetFilteringEnabled(false)
	model.Title = ""
	model.Select(0)

	for i, item := range items {
		if i > 0 {
			builder.WriteString("\n")
		}
		delegate.Render(&builder, model, i, item)
		if entry, ok := item.(historyListItem); ok {
			if strings.TrimSpace(entry.Description()) != "" && i < len(items)-1 {
				builder.WriteString("\n")
			}
		}
	}
	return strings.TrimRight(builder.String(), "\n")
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
