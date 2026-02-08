package editor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Adapter launches the configured editor.
type Adapter struct {
	Command string
}

// New returns an editor adapter using the given command.
func New(command string) *Adapter {
	return &Adapter{Command: command}
}

// Open launches the editor with the provided file path.
func (a Adapter) Open(path string) error {
	command := a.Command
	if command == "" {
		command = os.Getenv("VISUAL")
	}
	if command == "" {
		command = os.Getenv("EDITOR")
	}
	if command == "" {
		return errors.New("editor command is required")
	}
	cmd := exec.Command(command, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// OpenAtEnd opens a file and positions the cursor at the end when supported.
func (a Adapter) OpenAtEnd(path string) error {
	command := a.Command
	if command == "" {
		command = os.Getenv("VISUAL")
	}
	if command == "" {
		command = os.Getenv("EDITOR")
	}
	if command == "" {
		return errors.New("editor command is required")
	}
	if isVimEditor(command) {
		return openVimAtEnd(command, path)
	}
	return a.Open(path)
}

func isVimEditor(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	base := strings.ToLower(filepath.Base(fields[0]))
	return strings.Contains(base, "nvim") || strings.Contains(base, "vim") || base == "vi"
}

func openVimAtEnd(command, path string) error {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return errors.New("editor command is required")
	}
	args := append(fields[1:], "+normal G$", path)
	cmd := exec.Command(fields[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
