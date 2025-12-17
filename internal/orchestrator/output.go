package orchestrator

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/atotto/clipboard"
	"prompter-cli/internal/interfaces"
)

// OutputHandler implements the OutputHandler interface
type OutputHandler struct{}

// NewOutputHandler creates a new output handler
func NewOutputHandler() interfaces.OutputHandler {
	return &OutputHandler{}
}

// WriteToClipboard copies content to the system clipboard
func (h *OutputHandler) WriteToClipboard(content string) error {
	return clipboard.WriteAll(content)
}

// WriteToStdout writes content to standard output
func (h *OutputHandler) WriteToStdout(content string) error {
	_, err := fmt.Println(content)
	return err
}

// WriteToFile writes content to the specified file path
func (h *OutputHandler) WriteToFile(content string, path string) error {
	return ioutil.WriteFile(path, []byte(content), 0644)
}

// OpenInEditor opens content in the specified editor
func (h *OutputHandler) OpenInEditor(content string, editor string) error {
	// Create a temporary file
	tmpFile, err := ioutil.TempFile("", "prompter-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up

	// Write content to temporary file
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tmpFile.Close()

	// Launch editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to launch editor %s: %w", editor, err)
	}

	return nil
}