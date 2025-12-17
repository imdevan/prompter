package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"prompter-cli/internal/interfaces"
)

// Processor implements the TemplateProcessor interface
type Processor struct {
	promptsLocation string
}

// NewProcessor creates a new template processor
func NewProcessor(promptsLocation string) *Processor {
	return &Processor{
		promptsLocation: promptsLocation,
	}
}

// LoadTemplate loads a template from the specified path or discovers it by name
func (p *Processor) LoadTemplate(nameOrPath string) (*template.Template, error) {
	// If it's an absolute path or contains path separators, load directly
	if filepath.IsAbs(nameOrPath) || strings.Contains(nameOrPath, string(filepath.Separator)) {
		return p.loadTemplateFromPath(nameOrPath)
	}

	// Otherwise, discover the template by name (case-insensitive)
	templatePath, err := p.discoverTemplate(nameOrPath)
	if err != nil {
		return nil, err
	}

	return p.loadTemplateFromPath(templatePath)
}

// discoverTemplate finds a template file by name (case-insensitive matching by stem)
func (p *Processor) discoverTemplate(name string) (string, error) {
	// Check both pre and post directories
	directories := []string{
		filepath.Join(p.promptsLocation, "pre"),
		filepath.Join(p.promptsLocation, "post"),
	}

	for _, dir := range directories {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			// Get the file stem (filename without extension)
			filename := entry.Name()
			ext := filepath.Ext(filename)
			stem := strings.TrimSuffix(filename, ext)

			// Case-insensitive comparison
			if strings.EqualFold(stem, name) {
				return filepath.Join(dir, filename), nil
			}
		}
	}

	return "", fmt.Errorf("template not found: %s", name)
}

// loadTemplateFromPath loads a template from a specific file path
func (p *Processor) loadTemplateFromPath(path string) (*template.Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", path, err)
	}

	// Create template with custom delimiters and helper functions
	tmpl := template.New(filepath.Base(path))
	
	// Register helper functions before parsing
	if err := p.registerHelpersToTemplate(tmpl); err != nil {
		return nil, fmt.Errorf("failed to register helper functions: %w", err)
	}

	// Parse the template content
	tmpl, err = tmpl.Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", path, err)
	}

	return tmpl, nil
}

// Execute executes a template with the provided data
func (p *Processor) Execute(tmpl *template.Template, data interfaces.TemplateData) (string, error) {
	var buf strings.Builder
	
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RegisterHelpers registers custom template helper functions (placeholder for now)
func (p *Processor) RegisterHelpers() error {
	// This method is for global registration if needed
	// Individual templates get helpers registered in registerHelpersToTemplate
	return nil
}

// registerHelpersToTemplate registers both sprig and custom helper functions to a template
func (p *Processor) registerHelpersToTemplate(tmpl *template.Template) error {
	// Start with sprig functions
	funcMap := sprig.TxtFuncMap()
	
	// Add custom helper functions
	customFuncs := template.FuncMap{
		"truncate": truncateFunc,
		"mdFence":  mdFenceFunc,
		"indent":   indentFunc,
		"dedent":   dedentFunc,
	}
	
	// Merge custom functions into sprig functions
	for name, fn := range customFuncs {
		funcMap[name] = fn
	}
	
	// Apply the function map to the template
	tmpl.Funcs(funcMap)
	
	return nil
}

// truncateFunc truncates a string to a specified length
func truncateFunc(length int, text string) string {
	if len(text) <= length {
		return text
	}
	
	if length <= 3 {
		return text[:length]
	}
	
	return text[:length-3] + "..."
}

// mdFenceFunc wraps content in markdown fenced code blocks with optional language
func mdFenceFunc(language, content string) string {
	if language == "" {
		return fmt.Sprintf("```\n%s\n```", content)
	}
	return fmt.Sprintf("```%s\n%s\n```", language, content)
}

// indentFunc indents each line of text by the specified number of spaces
func indentFunc(spaces int, text string) string {
	if spaces <= 0 {
		return text
	}
	
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	
	for i, line := range lines {
		if strings.TrimSpace(line) != "" { // Don't indent empty lines
			lines[i] = indent + line
		}
	}
	
	return strings.Join(lines, "\n")
}

// dedentFunc removes common leading whitespace from all lines
func dedentFunc(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}
	
	// Find the minimum indentation (ignoring empty lines)
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		indent := 0
		for _, char := range line {
			if char == ' ' {
				indent++
			} else if char == '\t' {
				indent += 4 // Treat tab as 4 spaces
			} else {
				break
			}
		}
		
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	
	// If no indentation found, return original
	if minIndent <= 0 {
		return text
	}
	
	// Remove the common indentation
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		// Remove minIndent spaces/tabs from the beginning
		removed := 0
		for j, char := range line {
			if removed >= minIndent {
				lines[i] = line[j:]
				break
			}
			
			if char == ' ' {
				removed++
			} else if char == '\t' {
				removed += 4
				if removed > minIndent {
					// Partial tab removal - replace with spaces
					lines[i] = strings.Repeat(" ", removed-minIndent) + line[j+1:]
					break
				}
			} else {
				break
			}
		}
	}
	
	return strings.Join(lines, "\n")
}