package workflow

import (
	"os"
	"path/filepath"
	"strings"

	"prompter-cli/internal/utils"
)

func opencodeTemplateRoots() []string {
	roots := make([]string, 0, 2)
	if configHome := strings.TrimSpace(utils.XDGConfigHome()); configHome != "" {
		roots = append(roots, filepath.Join(configHome, "opencode"))
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		roots = append(roots, filepath.Join(home, ".opencode"))
	}
	return roots
}
