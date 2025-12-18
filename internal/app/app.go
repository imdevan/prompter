package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
