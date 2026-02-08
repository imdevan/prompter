package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"prompter-cli/internal/config"
	"prompter-cli/internal/ui"
)

func newFixCmd() *cobra.Command {
	opts := &rootOptions{}
	cfg := loadConfigForFlagRegistration()
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Generate a fix prompt from command output",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			manager := config.NewManager(cwd)
			cfg, err := manager.LoadWithOverride(opts.configPath)
			if err != nil {
				return err
			}
			theme := ui.ThemeFromConfig(cfg)
			piped, err := readStdinIfPiped(os.Stdin)
			if err != nil {
				return err
			}
			output := strings.TrimSpace(piped)
			if output == "" {
				lastCommand, err := lastShellCommand()
				if err != nil {
					return err
				}
				output, err = runCommandWithStatus(cmd.OutOrStdout(), lastCommand, theme)
				if err != nil {
					return err
				}
			}
			opts.fix = true
			opts.yes = true
			opts.interactive = false
			opts.fixOutput = output
			return runGenerate(cmd, opts, nil)
		},
	}

	addRootFlags(cmd, opts, cfg, rootFlagOptions{
		includeFix:     false,
		includeVersion: false,
	})
	registerTemplateFlags(cmd, opts, cfg, nil)
	setTemplateHelp(cmd)

	return cmd
}

type fixCommandResultMsg struct {
	output string
	err    error
}

type fixRunModel struct {
	command string
	spinner spinner.Model
	watch   stopwatch.Model
	output  string
	err     error
	done    bool
	theme   ui.Theme
}

func runCommandWithStatus(out io.Writer, command string, theme ui.Theme) (string, error) {
	model := newFixRunModel(command, theme)
	program := tea.NewProgram(model, tea.WithOutput(out), tea.WithoutSignalHandler())
	result, err := program.Run()
	if err != nil {
		return "", err
	}
	if m, ok := result.(fixRunModel); ok {
		if m.err != nil && strings.TrimSpace(m.output) == "" {
			return "", m.err
		}
		return m.output, nil
	}
	return "", errors.New("unexpected model result")
}

func newFixRunModel(command string, theme ui.Theme) fixRunModel {
	spin := spinner.New(spinner.WithSpinner(spinner.Line))
	spin.Style = lipgloss.NewStyle().Foreground(theme.Primary)
	return fixRunModel{
		command: command,
		spinner: spin,
		watch:   stopwatch.New(),
		theme:   theme,
	}
}

func (m fixRunModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.watch.Init(), runShellCommand(m.command))
}

func (m fixRunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	case fixCommandResultMsg:
		m.output = msg.output
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.watch, cmd = m.watch.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m fixRunModel) View() string {
	commandStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	timeStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	title := "Running previous command"
	if m.done {
		title = "Finished running previous command"
	}
	line := fmt.Sprintf("%s %s", m.spinner.View(), title)
	return strings.Join([]string{
		line,
		commandStyle.Render(m.command),
		timeStyle.Render("Elapsed: " + m.watch.View()),
	}, "\n")
}

func runShellCommand(command string) tea.Cmd {
	return func() tea.Msg {
		shell := os.Getenv("SHELL")
		if strings.TrimSpace(shell) == "" {
			shell = "/bin/sh"
		}
		cmd := exec.Command(shell, "-lc", command)
		out, err := cmd.CombinedOutput()
		return fixCommandResultMsg{output: string(out), err: err}
	}
}

func lastShellCommand() (string, error) {
	shell := os.Getenv("SHELL")
	if strings.TrimSpace(shell) == "" {
		shell = "/bin/sh"
	}

	last, err := historyLookup(shell, "fc -ln -1")
	if err != nil || last == "" {
		last, _ = historyLookup(shell, "history -p !!")
	}
	last = strings.TrimSpace(last)
	if last == "" {
		last = strings.TrimSpace(lastCommandFromHistoryFile(shell))
	}
	if strings.HasPrefix(last, "prompter fix") {
		return "", errors.New("previous command is prompter fix; try piping output into prompter fix")
	}
	if last == "" {
		return "", errors.New("unable to determine previous command; try piping output into prompter fix")
	}
	return last, nil
}

func historyLookup(shell, query string) (string, error) {
	cmd := exec.Command(shell, "-lc", query)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func lastCommandFromHistoryFile(shell string) string {
	historyPath := os.Getenv("HISTFILE")
	if historyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		switch filepath.Base(shell) {
		case "zsh":
			historyPath = filepath.Join(home, ".zsh_history")
		case "bash":
			historyPath = filepath.Join(home, ".bash_history")
		default:
			historyPath = filepath.Join(home, ".bash_history")
		}
	}

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if filepath.Base(shell) == "zsh" && strings.HasPrefix(line, ":") {
			if idx := strings.Index(line, ";"); idx != -1 && idx+1 < len(line) {
				line = strings.TrimSpace(line[idx+1:])
			}
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "prompter fix") {
			continue
		}
		return line
	}
	return ""
}
