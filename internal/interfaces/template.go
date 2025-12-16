package interfaces

import (
	"text/template"
	"time"
)

// TemplateData contains all variables available to templates
type TemplateData struct {
	Prompt string                 `json:"prompt"`
	Now    time.Time              `json:"now"`
	CWD    string                 `json:"cwd"`
	Files  []FileInfo             `json:"files"`
	Git    GitInfo                `json:"git"`
	Config map[string]interface{} `json:"config"`
	Env    map[string]string      `json:"env"`
	Fix    FixInfo                `json:"fix"`
}

// FileInfo represents information about a file for templates
type FileInfo struct {
	Path     string `json:"path"`
	RelPath  string `json:"rel_path"`
	Language string `json:"language"`
	Content  string `json:"content"`
}

// GitInfo represents git repository information
type GitInfo struct {
	Root   string `json:"root"`
	Branch string `json:"branch"`
	Commit string `json:"commit"`
	Dirty  bool   `json:"dirty"`
}

// FixInfo represents fix mode data
type FixInfo struct {
	Enabled bool   `json:"enabled"`
	Raw     string `json:"raw"`
	Command string `json:"command"`
	Output  string `json:"output"`
}

// TemplateProcessor handles template loading and execution
type TemplateProcessor interface {
	// LoadTemplate loads a template from the specified path
	LoadTemplate(path string) (*template.Template, error)
	
	// Execute executes a template with the provided data
	Execute(tmpl *template.Template, data TemplateData) (string, error)
	
	// RegisterHelpers registers custom template helper functions
	RegisterHelpers() error
}