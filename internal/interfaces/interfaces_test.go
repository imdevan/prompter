package interfaces

import (
	"testing"
	"text/template"
	"time"
)

// Test that all interfaces can be implemented (compilation test)
func TestInterfaceCompilation(t *testing.T) {
	// Test that we can create instances of all data structures
	config := &Config{
		PromptsLocation:   "/test",
		Editor:            "vim",
		DirectoryStrategy: "git",
		Target:            "clipboard",
	}

	templateData := &TemplateData{
		Prompt: "test prompt",
		Now:    time.Now(),
		CWD:    "/test/dir",
		Files:  []FileInfo{},
		Git:    GitInfo{},
		Config: make(map[string]interface{}),
		Env:    make(map[string]string),
		Fix:    FixInfo{},
	}

	fileInfo := &FileInfo{
		Path:     "/test/file.go",
		RelPath:  "file.go",
		Language: "go",
		Content:  "package main",
	}

	// Verify structs are properly defined
	if config == nil || templateData == nil || fileInfo == nil {
		t.Error("Failed to create interface data structures")
	}
}

// Mock implementations to verify interfaces are properly defined
type mockConfigManager struct{}

func (m *mockConfigManager) Load(path string) (*Config, error) {
	return &Config{}, nil
}

func (m *mockConfigManager) Resolve() (*Config, error) {
	return &Config{}, nil
}

func (m *mockConfigManager) Validate(config *Config) error {
	return nil
}

type mockTemplateProcessor struct{}

func (m *mockTemplateProcessor) LoadTemplate(path string) (*template.Template, error) {
	return template.New("test"), nil
}

func (m *mockTemplateProcessor) Execute(tmpl *template.Template, data TemplateData) (string, error) {
	return "test output", nil
}

func (m *mockTemplateProcessor) RegisterHelpers() error {
	return nil
}



type mockOutputHandler struct{}

func (m *mockOutputHandler) WriteToClipboard(content string) error {
	return nil
}

func (m *mockOutputHandler) WriteToStdout(content string) error {
	return nil
}

func (m *mockOutputHandler) WriteToFile(content string, path string) error {
	return nil
}

func (m *mockOutputHandler) OpenInEditor(content string, editor string) error {
	return nil
}

// Test that mock implementations satisfy interfaces
func TestInterfaceImplementations(t *testing.T) {
	var _ ConfigManager = &mockConfigManager{}
	var _ TemplateProcessor = &mockTemplateProcessor{}
	var _ OutputHandler = &mockOutputHandler{}
}