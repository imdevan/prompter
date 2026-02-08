package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompter-cli/internal/testutil"
)

func TestFixturesFrontmatter(t *testing.T) {
	root, err := findFixturesRoot()
	if err != nil {
		t.Fatalf("locate fixtures: %v", err)
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !shouldValidateFixtureFrontmatter(rel) {
			return nil
		}
		if strings.HasPrefix(rel, ".agents/skills/") || strings.HasPrefix(rel, ".opencode/skills/") {
			if entry.Name() != "SKILL.md" {
				t.Errorf("expected SKILL.md for skill fixture, got %q", rel)
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		header, _, ok := splitFrontmatter(string(data))
		if !ok {
			t.Errorf("missing frontmatter: %s", rel)
			return nil
		}
		meta := frontmatterMap(header)
		if strings.TrimSpace(meta["name"]) == "" {
			t.Errorf("missing frontmatter name: %s", rel)
		}
		if strings.TrimSpace(meta["title"]) != "" {
			t.Errorf("legacy title frontmatter not allowed: %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixtures: %v", err)
	}
}

func shouldValidateFixtureFrontmatter(rel string) bool {
	switch {
	case rel == "agents.md":
		return true
	case strings.HasPrefix(rel, "prompts/"):
		return true
	case strings.HasPrefix(rel, "cursor/commands/"):
		return true
	case strings.HasPrefix(rel, "kiro/steering/"):
		return true
	case strings.HasPrefix(rel, ".agents/skills/"):
		return true
	case strings.HasPrefix(rel, ".opencode/skills/"):
		return true
	default:
		return false
	}
}

func frontmatterMap(header string) map[string]string {
	meta := make(map[string]string)
	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(strings.Trim(value, "\""))
		meta[key] = value
	}
	return meta
}

func findFixturesRoot() (string, error) {
	start, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := start
	for {
		candidate := filepath.Join(dir, testutil.FixturesPath())
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}
