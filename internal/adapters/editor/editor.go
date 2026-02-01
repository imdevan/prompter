package editor

import (
	"errors"
	"os"
	"os/exec"
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
