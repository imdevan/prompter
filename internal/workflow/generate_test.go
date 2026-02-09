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

func TestGeneratorIncludeAgents(t *testing.T) {
	root := t.TempDir()
	templatesDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	agentsContent := "---\n" +
		"name: Agent Instructions\n" +
		"---\n" +
		"Agent guidance"
	if err := os.WriteFile(agentsPath, []byte(agentsContent), 0o644); err != nil {
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
	if strings.Contains(out, "name: Agent Instructions") {
		t.Fatalf("expected frontmatter stripped from agents content, got %q", out)
	}
}

func TestGeneratorStripsTemplateFrontmatter(t *testing.T) {
	root := t.TempDir()
	templatesDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	templateContent := "---\n" +
		"name: Question\n" +
		"description: Question - no code no output\n" +
		"pin: true\n" +
		"---\n" +
		"\n" +
		"The following is a question. No Code.\n"
	templatePath := filepath.Join(templatesDir, "question.md")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	repo := template.NewRepository(templatesDir)
	gen := NewGenerator(repo)
	req := domain.Request{
		BasePrompt:    "Does frontmatter get stripped?",
		TemplateNames: []string{"question"},
	}

	out, err := gen.Run(req, domain.DefaultConfig())
	if err != nil {
		t.Fatalf("run generator: %v", err)
	}
	if !strings.Contains(out, "The following is a question. No Code.") {
		t.Fatalf("expected template content, got %q", out)
	}
	if strings.Contains(out, "name: Question") {
		t.Fatalf("expected frontmatter stripped from template content, got %q", out)
	}
}

func TestGeneratorRightToLeftTemplatePipeline(t *testing.T) {
	root := t.TempDir()
	templatesDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	writeTemplate := func(name, content string) {
		t.Helper()
		path := filepath.Join(templatesDir, name+".md")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write template %s: %v", name, err)
		}
	}

	writeTemplate("question", "Question")
	writeTemplate("test", "Test")
	writeTemplate("validate", "{{.Prompt}}\nValidate")

	repo := template.NewRepository(templatesDir)
	gen := NewGenerator(repo)
	req := domain.Request{
		BasePrompt:    "Base",
		TemplateNames: []string{"question", "test", "validate"},
	}

	out, err := gen.Run(req, domain.DefaultConfig())
	if err != nil {
		t.Fatalf("run generator: %v", err)
	}
	if out != "Question\n\nTest\n\nBase\n\nValidate" {
		t.Fatalf("unexpected prompt order: %q", out)
	}
}

func TestGeneratorWrapperTemplateWrapsPrompt(t *testing.T) {
	root := t.TempDir()
	templatesDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	writeTemplate := func(name, content string) {
		t.Helper()
		path := filepath.Join(templatesDir, name+".md")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write template %s: %v", name, err)
		}
	}

	writeTemplate("question", "Question")
	writeTemplate("wrapper", "Begin\n{{.Prompt}}\nEnd")
	writeTemplate("test", "Test")
	writeTemplate("validate", "{{.Prompt}}\nValidate")

	repo := template.NewRepository(templatesDir)
	gen := NewGenerator(repo)
	req := domain.Request{
		BasePrompt:    "Base",
		TemplateNames: []string{"question", "wrapper", "test", "validate"},
	}

	out, err := gen.Run(req, domain.DefaultConfig())
	if err != nil {
		t.Fatalf("run generator: %v", err)
	}
	expected := strings.Join([]string{
		"Question",
		"Begin",
		"Test",
		"Base",
		"Validate",
		"End",
	}, "\n\n")
	if out != expected {
		t.Fatalf("unexpected prompt order: %q", out)
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
