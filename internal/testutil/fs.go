package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"prompter-cli/internal/domain"
)

// EnsureDirs creates the configured prompts/history directories for tests.
func EnsureDirs(t *testing.T, cfg domain.Config) {
	t.Helper()
	dirs := []string{cfg.PromptsLocation, cfg.HistoryLocation}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
}

// WithTempWorkspace sets CWD to a temporary directory.
func WithTempWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return root
}

// LocalPromptsPath resolves the local prompts directory for a workspace.
func LocalPromptsPath(cwd string, cfg domain.Config) string {
	if cfg.LocalPromptsLocation == "" {
		return ""
	}
	return filepath.Join(cwd, cfg.LocalPromptsLocation)
}
