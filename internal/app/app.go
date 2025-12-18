package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"prompter-cli/internal/interactive"
	"prompter-cli/internal/interfaces"
	"prompter-cli/internal/orchestrator"
	"prompter-cli/pkg/models"
)

// Run executes the main application logic
func Run(request *models.PromptRequest) error {
	// Create orchestrator first to load configuration
	orch := orchestrator.New()

	// Load configuration to get the correct prompts location
	cfg, err := orch.LoadConfiguration(request.ConfigPath)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Resolve interactive mode based on flags and config
	resolveInteractiveMode(request, cfg)

	// Create interactive prompter with the configured prompts location
	prompter := interactive.NewPrompter(cfg.PromptsLocation)

	// Collect missing inputs interactively if needed
	if err := prompter.CollectMissingInputs(request); err != nil {
		return fmt.Errorf("failed to collect inputs: %w", err)
	}

	// Generate the prompt
	prompt, err := orch.GeneratePrompt(request)
	if err != nil {
		return fmt.Errorf("prompt generation failed: %w", err)
	}

	// Output the prompt
	if err := orch.OutputPrompt(prompt, request, cfg); err != nil {
		return fmt.Errorf("output failed: %w", err)
	}

	return nil
}

// resolveInteractiveMode determines the final interactive mode based on flags and config
func resolveInteractiveMode(request *models.PromptRequest, cfg *interfaces.Config) {
	// Priority: explicit flags > config default
	if request.ForceInteractive {
		request.Interactive = true
	} else if request.ForceNonInteractive {
		request.Interactive = false
	} else {
		// Use config default
		request.Interactive = cfg.InteractiveDefault
	}
}

// getDefaultPromptsLocation returns the default prompts location
func getDefaultPromptsLocation() string {
	// Try to get from current working directory first
	if cwd, err := os.Getwd(); err == nil {
		promptsDir := filepath.Join(cwd, "prompts")
		if _, err := os.Stat(promptsDir); err == nil {
			return promptsDir
		}
	}

	// Fallback to home directory
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "prompter", "prompts")
	}

	// Final fallback
	return "prompts"
}

// ListTemplates lists all available prompt templates
func ListTemplates(request *models.PromptRequest) error {
	// Create orchestrator to load configuration
	orch := orchestrator.New()

	// Load configuration to get the prompts location
	cfg, err := orch.LoadConfiguration(request.ConfigPath)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Display prompts location with ~ for home directory
	displayPath := contractPath(cfg.PromptsLocation)
	fmt.Printf("Prompts location: %s\n\n", displayPath)

	// List pre-templates
	preDir := filepath.Join(cfg.PromptsLocation, "pre")
	preTemplates, err := listTemplatesInDir(preDir)
	if err != nil {
		fmt.Printf("Pre-templates: (directory not found)\n")
	} else if len(preTemplates) == 0 {
		fmt.Printf("Pre-templates: (none found)\n")
	} else {
		fmt.Printf("Pre-templates:\n")
		for _, tmpl := range preTemplates {
			fmt.Printf("  - %s\n", tmpl)
		}
	}

	fmt.Println()

	// List post-templates
	postDir := filepath.Join(cfg.PromptsLocation, "post")
	postTemplates, err := listTemplatesInDir(postDir)
	if err != nil {
		fmt.Printf("Post-templates: (directory not found)\n")
	} else if len(postTemplates) == 0 {
		fmt.Printf("Post-templates: (none found)\n")
	} else {
		fmt.Printf("Post-templates:\n")
		for _, tmpl := range postTemplates {
			fmt.Printf("  - %s\n", tmpl)
		}
	}

	return nil
}

// listTemplatesInDir lists all .md files in a directory
func listTemplatesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var templates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Only include .md files
		if filepath.Ext(name) == ".md" {
			// Remove .md extension and .default. prefix if present
			templateName := name[:len(name)-3] // Remove .md

			// Remove .default. prefix if present
			if len(templateName) > 9 && templateName[:9] == ".default." {
				templateName = templateName[9:]
			}

			templates = append(templates, templateName)
		}
	}

	return templates, nil
}

// contractPath converts a full path back to use ~ for the home directory
func contractPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path // Return original path if we can't get home dir
	}

	// Add trailing slash to home directory for proper matching
	homeDirWithSlash := homeDir + string(filepath.Separator)
	pathWithSlash := path + string(filepath.Separator)

	// Check if path starts with home directory
	if strings.HasPrefix(pathWithSlash, homeDirWithSlash) {
		// Replace home directory with ~
		relativePath := path[len(homeDir):]
		if relativePath == "" {
			return "~"
		}
		return "~" + relativePath
	}

	return path
}
// AddTemplate adds a new prompt template
func AddTemplate(request *models.PromptRequest, content, preName, postName string, fromClipboard bool) error {
	// Create orchestrator to load configuration
	orch := orchestrator.New()

	// Load configuration to get the prompts location
	cfg, err := orch.LoadConfiguration(request.ConfigPath)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Resolve interactive mode based on flags and config
	resolveInteractiveMode(request, cfg)

	// Determine template type and name
	var templateType, templateName string
	
	// Check if both pre and post flags are provided (invalid)
	if preName != "" && postName != "" {
		return fmt.Errorf("cannot specify both --pre and --post flags")
	}
	
	// If interactive mode is forced with -i, always go interactive regardless of flags
	if request.ForceInteractive {
		// Interactive mode - ask user for template type and name
		prompter := interactive.NewPrompter(cfg.PromptsLocation)
		templateType, templateName, err = prompter.CollectTemplateInfo()
		if err != nil {
			return fmt.Errorf("failed to collect template information: %w", err)
		}
	} else if preName != "" {
		templateType = "pre"
		templateName = preName
	} else if postName != "" {
		templateType = "post"
		templateName = postName
	} else if !request.Interactive {
		return fmt.Errorf("must specify either --pre or --post flag in non-interactive mode")
	} else {
		// Interactive mode - ask user for template type and name
		prompter := interactive.NewPrompter(cfg.PromptsLocation)
		templateType, templateName, err = prompter.CollectTemplateInfo()
		if err != nil {
			return fmt.Errorf("failed to collect template information: %w", err)
		}
	}

	// Get content
	var templateContent string
	if fromClipboard {
		// Get content from clipboard
		templateContent, err = getClipboardContent()
		if err != nil {
			return fmt.Errorf("failed to get clipboard content: %w", err)
		}
	} else if content != "" {
		templateContent = content
	} else if !request.Interactive {
		return fmt.Errorf("must provide content as argument or use --clipboard flag in non-interactive mode")
	} else {
		// Interactive mode - ask user for content
		prompter := interactive.NewPrompter(cfg.PromptsLocation)
		templateContent, err = prompter.CollectTemplateContent()
		if err != nil {
			return fmt.Errorf("failed to collect template content: %w", err)
		}
	}

	// Create the template file
	templateDir := filepath.Join(cfg.PromptsLocation, templateType)
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	templatePath := filepath.Join(templateDir, templateName+".md")
	
	// Check if file already exists
	if _, err := os.Stat(templatePath); err == nil {
		if request.Interactive {
			prompter := interactive.NewPrompter(cfg.PromptsLocation)
			overwrite, err := prompter.ConfirmOverwrite(templatePath)
			if err != nil {
				return fmt.Errorf("failed to get overwrite confirmation: %w", err)
			}
			if !overwrite {
				fmt.Println("Template creation cancelled.")
				return nil
			}
		} else {
			return fmt.Errorf("template file already exists: %s", contractPath(templatePath))
		}
	}

	// Write the template file
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	fmt.Printf("Created %s template: %s\n", templateType, contractPath(templatePath))
	return nil
}

// getClipboardContent gets content from the system clipboard
func getClipboardContent() (string, error) {
	content, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to read from clipboard: %w", err)
	}
	
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("clipboard is empty")
	}
	
	return content, nil
}