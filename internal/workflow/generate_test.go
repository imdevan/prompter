package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
)

func TestGeneratorRunWithTemplates(t *testing.T) {
	root := t.TempDir()
	templatesDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	fixturesRoot := filepath.Join("..", "..", "tests", "templates")
	if err := copyFile(filepath.Join(fixturesRoot, "global", "index.md"), filepath.Join(templatesDir, "index.md")); err != nil {
		t.Fatalf("copy index: %v", err)
	}
	if err := copyFile(filepath.Join(fixturesRoot, "local", "question.md"), filepath.Join(templatesDir, "question.md")); err != nil {
		t.Fatalf("copy question: %v", err)
	}

	filePath := filepath.Join(root, "example.txt")
	if err := os.WriteFile(filePath, []byte("hello file"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	repo := template.NewRepository(templatesDir)
	gen := NewGenerator(repo)
	req := domain.Request{
		BasePrompt:    "Explain this output.",
		TemplateNames: []string{"question"},
		Files:         []string{filePath},
	}

	out, err := gen.Run(req, domain.DefaultConfig())
	if err != nil {
		t.Fatalf("run generator: %v", err)
	}
	if !strings.Contains(out, "Hello from index.") {
		t.Fatalf("expected index template content, got %q", out)
	}
	if !strings.Contains(out, "# Question") {
		t.Fatalf("expected question template content, got %q", out)
	}
	if !strings.Contains(out, "Explain this output.") {
		t.Fatalf("expected base prompt content, got %q", out)
	}
	if !strings.Contains(out, "Files:") || !strings.Contains(out, "hello file") {
		t.Fatalf("expected file content inclusion, got %q", out)
	}
}

func TestGeneratorRunFixModeDefaultTemplate(t *testing.T) {
	root := t.TempDir()
	repo := template.NewRepository(root)
	gen := NewGenerator(repo)
	req := domain.Request{
		Fix: domain.FixInput{
			Enabled: true,
			Output:  "failing test output",
		},
	}

	out, err := gen.Run(req, domain.DefaultConfig())
	if err != nil {
		t.Fatalf("run generator: %v", err)
	}
	if !strings.Contains(out, "Please evaluate the following output for errors") {
		t.Fatalf("expected default fix template content, got %q", out)
	}
	if !strings.Contains(out, "failing test output") {
		t.Fatalf("expected fix output inclusion, got %q", out)
	}
}

func TestGeneratorIncludeAgents(t *testing.T) {
	root := t.TempDir()
	templatesDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("Agent guidance"), 0o644); err != nil {
		t.Fatalf("write agents: %v", err)
	}
	repo := template.NewRepository(templatesDir)
	gen := NewGenerator(repo)
	req := domain.Request{
		BasePrompt: "Check agent content.",
		CWD:        root,
	}
	req.TemplateNames = append(req.TemplateNames, "agents.md")

	out, err := gen.Run(req, domain.DefaultConfig())
	if err != nil {
		t.Fatalf("run generator: %v", err)
	}
	if !strings.Contains(out, "Agent guidance") {
		t.Fatalf("expected agents content, got %q", out)
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
