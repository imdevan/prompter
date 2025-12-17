package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"prompter-cli/internal/interfaces"
)

func TestProcessor_IntegrationWithRealTemplates(t *testing.T) {
	// Use the actual prompts directory
	promptsDir := filepath.Join("..", "..", "..", "prompts")
	processor := NewProcessor(promptsDir)
	
	// Test loading a real template
	tmpl, err := processor.LoadTemplate("software-architect")
	if err != nil {
		t.Fatalf("failed to load real template: %v", err)
	}
	
	// Test executing with sample data
	data := interfaces.TemplateData{
		Prompt: "Create a user authentication system",
		Now:    time.Now(),
		CWD:    "/project/root",
		Files: []interfaces.FileInfo{
			{
				Path:     "/project/root/main.go",
				RelPath:  "main.go",
				Language: "go",
				Content:  "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
			},
		},
		Git: interfaces.GitInfo{
			Root:   "/project/root",
			Branch: "main",
			Commit: "abc123",
			Dirty:  false,
		},
		Config: map[string]interface{}{
			"max_file_size": 1024,
		},
		Env: map[string]string{
			"USER": "testuser",
		},
		Fix: interfaces.FixInfo{
			Enabled: false,
		},
	}
	
	result, err := processor.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	
	// Verify the result contains expected content
	if !strings.Contains(result, "LEAD SOFTWARE ARCHITECT") {
		t.Errorf("expected template content not found in result")
	}
	
	// Verify it's not empty
	if strings.TrimSpace(result) == "" {
		t.Errorf("template execution resulted in empty output")
	}
}

func TestProcessor_TemplateWithVariables(t *testing.T) {
	// Create a temporary template with variables
	tempDir := t.TempDir()
	preDir := filepath.Join(tempDir, "pre")
	if err := os.MkdirAll(preDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	templateContent := `# Project: {{.Prompt}}

Current directory: {{.CWD}}
Current time: {{.Now.Format "2006-01-02 15:04:05"}}

## Files
{{range .Files}}
### {{.RelPath}} ({{.Language}})
{{mdFence .Language .Content}}
{{end}}

## Git Info
- Branch: {{.Git.Branch}}
- Commit: {{.Git.Commit}}
- Dirty: {{.Git.Dirty}}

## Configuration
{{range $key, $value := .Config}}
- {{$key}}: {{$value}}
{{end}}

## Environment
User: {{.Env.USER}}

{{if .Fix.Enabled}}
## Fix Mode
Command: {{.Fix.Command}}
Output:
{{mdFence "text" .Fix.Output}}
{{end}}`
	
	templatePath := filepath.Join(preDir, "test-vars.md")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	processor := NewProcessor(tempDir)
	tmpl, err := processor.LoadTemplate("test-vars")
	if err != nil {
		t.Fatalf("failed to load template: %v", err)
	}
	
	data := interfaces.TemplateData{
		Prompt: "Test Project",
		Now:    time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC),
		CWD:    "/test/project",
		Files: []interfaces.FileInfo{
			{
				Path:     "/test/project/main.go",
				RelPath:  "main.go",
				Language: "go",
				Content:  "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
			},
		},
		Git: interfaces.GitInfo{
			Root:   "/test/project",
			Branch: "feature/test",
			Commit: "def456",
			Dirty:  true,
		},
		Config: map[string]interface{}{
			"max_file_size": 2048,
			"debug":         true,
		},
		Env: map[string]string{
			"USER": "testuser",
			"HOME": "/home/testuser",
		},
		Fix: interfaces.FixInfo{
			Enabled: true,
			Command: "go build",
			Output:  "build failed: syntax error",
		},
	}
	
	result, err := processor.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	
	// Verify various parts of the template were processed correctly
	expectedParts := []string{
		"# Project: Test Project",
		"Current directory: /test/project",
		"Current time: 2023-12-25 10:30:00",
		"### main.go (go)",
		"```go",
		"package main",
		"- Branch: feature/test",
		"- Commit: def456",
		"- Dirty: true",
		"- max_file_size: 2048",
		"- debug: true",
		"User: testuser",
		"## Fix Mode",
		"Command: go build",
		"```text",
		"build failed: syntax error",
	}
	
	for _, expected := range expectedParts {
		if !strings.Contains(result, expected) {
			t.Errorf("expected %q to be in result, but it wasn't found", expected)
		}
	}
}