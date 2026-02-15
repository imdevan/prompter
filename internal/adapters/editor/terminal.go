package editor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func IsEmacs(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	base := strings.ToLower(filepath.Base(fields[0]))
	return strings.Contains(base, "emacs")
}

func IsNano(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	base := strings.ToLower(filepath.Base(fields[0]))
	return strings.Contains(base, "nano")
}

func OpenEmacsAtLine(command, path string, line int) error {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return errors.New("editor command is required")
	}
	args := append(fields[1:], fmt.Sprintf("+%d", line), path)
	return runEditorCommand(fields[0], args)
}

func OpenNanoAtLine(command, path string, line int) error {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return errors.New("editor command is required")
	}

	args := append(fields[1:], fmt.Sprintf("+%d", line), path)
	return runEditorCommand(fields[0], args)
}

func OpenEmacsAtEnd(command, path string) error {
	line, err := countFileLines(path)
	if err != nil {
		return err
	}
	return OpenEmacsAtLine(command, path, line)
}

func OpenNanoAtEnd(command, path string) error {
	line, err := countFileLines(path)
	if err != nil {
		return err
	}
	return OpenNanoAtLine(command, path, line)
}

func countFileLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 1, nil
	}
	return bytes.Count(data, []byte{'\n'}) + 1, nil
}
