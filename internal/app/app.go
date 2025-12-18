package app

import (
	"fmt"
	"os"
	"path/filepath"

	"prompter-cli/internal/interfaces"
	"prompter-cli/internal/interactive"
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

