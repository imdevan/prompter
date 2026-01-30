package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryListAndGet(t *testing.T) {
	root := t.TempDir()
	globalDir := filepath.Join(root, "global")
	localDir := filepath.Join(root, "local")

	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatalf("mkdir local: %v", err)
	}

	fixturesRoot := filepath.Join("..", "..", "tests", "templates")
	if err := copyFile(filepath.Join(fixturesRoot, "global", "index.md"), filepath.Join(globalDir, "index.md")); err != nil {
		t.Fatalf("copy index: %v", err)
	}
	if err := copyFile(filepath.Join(fixturesRoot, "global", "fix.md"), filepath.Join(globalDir, "fix.md")); err != nil {
		t.Fatalf("copy fix: %v", err)
	}
	if err := copyFile(filepath.Join(fixturesRoot, "local", "question.md"), filepath.Join(localDir, "question.md")); err != nil {
		t.Fatalf("copy question: %v", err)
	}

	repo := NewRepository(localDir, globalDir)
	templates, err := repo.List()
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if len(templates) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(templates))
	}

	got, err := repo.Get("question")
	if err != nil {
		t.Fatalf("get question: %v", err)
	}
	if got.Title != "Question Template" {
		t.Fatalf("expected title from frontmatter, got %q", got.Title)
	}
	if got.Flag != "question" || got.Shorthand != "q" {
		t.Fatalf("expected flag/shorthand from frontmatter, got %q/%q", got.Flag, got.Shorthand)
	}
	if got.Location != localDir {
		t.Fatalf("expected local location, got %q", got.Location)
	}

	index, err := repo.Get("index")
	if err != nil {
		t.Fatalf("get index: %v", err)
	}
	if !index.Pinned {
		t.Fatalf("expected index to be pinned")
	}
	if index.Location != globalDir {
		t.Fatalf("expected global location, got %q", index.Location)
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
