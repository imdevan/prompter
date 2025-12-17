package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"prompter-cli/internal/config"
	"prompter-cli/internal/interfaces"
	"prompter-cli/internal/template"
	"prompter-cli/pkg/models"
)

// Orchestrator coordinates all components to generate prompts
type Orchestrator struct {
	configManager     interfaces.ConfigManager
	templateProcessor interfaces.TemplateProcessor
	outputHandler     interfaces.OutputHandler
}

// New creates a new orchestrator with all required components
func New() *Orchestrator {
	return &Orchestrator{
		configManager:     config.NewManager(),
		templateProcessor: template.NewProcessor(""),
		outputHandler:     NewOutputHandler(), // We'll implement this
	}
}

// GeneratePrompt orchestrates the entire prompt generation process
func (o *Orchestrator) GeneratePrompt(request *models.PromptRequest) (string, error) {
	// Validate request first
	if err := o.validateRequest(request); err != nil {
		return "", RecoverFromError(err)
	}

	// Load and resolve configuration
	cfg, err := o.loadConfiguration(request.ConfigPath)
	if err != nil {
		configErr := NewConfigurationError("failed to load configuration", err)
		return "", RecoverFromError(configErr)
	}

	// Apply configuration defaults to request
	o.applyConfigDefaults(request, cfg)

	// Detect and handle mode (normal vs fix)
	if request.FixMode {
		return o.generateFixModePrompt(request, cfg)
	}

	return o.generateNormalPrompt(request, cfg)
}

// LoadConfiguration loads and resolves configuration with precedence (exported for app layer)
func (o *Orchestrator) LoadConfiguration(configPath string) (*interfaces.Config, error) {
	return o.loadConfiguration(configPath)
}

// loadConfiguration loads and resolves configuration with precedence
func (o *Orchestrator) loadConfiguration(configPath string) (*interfaces.Config, error) {
	// Load configuration from file first
	_, err := o.configManager.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply precedence resolution
	cfg, err := o.configManager.Resolve()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve configuration: %w", err)
	}

	// Validate configuration
	if err := o.configManager.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// applyConfigDefaults applies configuration defaults to the request
func (o *Orchestrator) applyConfigDefaults(request *models.PromptRequest, cfg *interfaces.Config) {
	if request.PreTemplate == "" && cfg.DefaultPre != "" {
		request.PreTemplate = cfg.DefaultPre
	}
	if request.PostTemplate == "" && cfg.DefaultPost != "" {
		request.PostTemplate = cfg.DefaultPost
	}
	if request.Target == "" && cfg.Target != "" {
		request.Target = cfg.Target
	}
	// Don't set editor from config - only use when explicitly requested
	if request.FixFile == "" && cfg.FixFile != "" {
		request.FixFile = cfg.FixFile
	}
}

// generateNormalPrompt generates a prompt in normal mode
func (o *Orchestrator) generateNormalPrompt(request *models.PromptRequest, cfg *interfaces.Config) (string, error) {
	var promptParts []string

	// Process pre-template if specified
	if request.PreTemplate != "" {
		preContent, err := o.processTemplate(request.PreTemplate, request, cfg, "pre")
		if err != nil {
			templateErr := NewTemplateError(request.PreTemplate, err)
			// Check if this is recoverable (template not found)
			if IsRecoverableError(templateErr) {
				// Log warning but continue without template
				fmt.Fprintf(os.Stderr, "Warning: %s\n", templateErr.Error())
			} else {
				return "", RecoverFromError(templateErr)
			}
		} else if preContent != "" {
			promptParts = append(promptParts, preContent)
		}
	}

	// Add base prompt
	if request.BasePrompt != "" {
		promptParts = append(promptParts, request.BasePrompt)
	}

	// Include file content
	if len(request.Files) > 0 || request.Directory != "" {
		contentPart := o.formatContent(request)
		if contentPart != "" {
			promptParts = append(promptParts, contentPart)
		}
	}

	// Process post-template if specified
	if request.PostTemplate != "" {
		postContent, err := o.processTemplate(request.PostTemplate, request, cfg, "post")
		if err != nil {
			templateErr := NewTemplateError(request.PostTemplate, err)
			// Check if this is recoverable (template not found)
			if IsRecoverableError(templateErr) {
				// Log warning but continue without template
				fmt.Fprintf(os.Stderr, "Warning: %s\n", templateErr.Error())
			} else {
				return "", RecoverFromError(templateErr)
			}
		} else if postContent != "" {
			promptParts = append(promptParts, postContent)
		}
	}

	return strings.Join(promptParts, "\n\n"), nil
}

// generateFixModePrompt generates a prompt in fix mode
func (o *Orchestrator) generateFixModePrompt(request *models.PromptRequest, cfg *interfaces.Config) (string, error) {
	// Load fix content
	fixContent, err := o.loadFixContent(request.FixFile)
	if err != nil {
		fixErr := NewFixModeError(request.FixFile, err)
		return "", RecoverFromError(fixErr)
	}

	var promptParts []string

	// Try to load fix.md template, fallback to default
	fixTemplate := "fix"
	preContent, err := o.processTemplate(fixTemplate, request, cfg, "pre")
	if err != nil {
		// Use default fix template content (graceful fallback)
		preContent = "Please help me fix this command that failed:\n"
	}
	if preContent != "" {
		promptParts = append(promptParts, preContent)
	}

	// Add fix content wrapped in markdown code block
	wrappedContent := fmt.Sprintf("```text\n%s\n```", fixContent)
	promptParts = append(promptParts, wrappedContent)

	return strings.Join(promptParts, "\n\n"), nil
}

// processTemplate processes a template with the current context
func (o *Orchestrator) processTemplate(templateName string, request *models.PromptRequest, cfg *interfaces.Config, templateType string) (string, error) {
	// Update template processor with prompts location
	if processor, ok := o.templateProcessor.(*template.Processor); ok {
		processor.SetPromptsLocation(cfg.PromptsLocation)
	}

	// Build template path
	templatePath := filepath.Join(cfg.PromptsLocation, templateType, templateName+".md")

	// Load template
	tmpl, err := o.templateProcessor.LoadTemplate(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", templateName, err)
	}

	// Build template data
	templateData, err := o.buildTemplateData(request, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to build template data: %w", err)
	}

	// Execute template
	result, err := o.templateProcessor.Execute(tmpl, *templateData)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return result, nil
}

// formatContent formats files and directory for inclusion in the prompt
func (o *Orchestrator) formatContent(request *models.PromptRequest) string {
	var parts []string

	// Add file references
	if len(request.Files) > 0 {
		parts = append(parts, "Referencing files:")
		for _, file := range request.Files {
			parts = append(parts, file)
		}
	}

	// Add directory reference using current working directory
	if request.Directory != "" {
		parts = append(parts, "Referencing dir:")
		if request.Directory == "." {
			if cwd, err := os.Getwd(); err == nil {
				parts = append(parts, cwd)
			} else {
				parts = append(parts, request.Directory)
			}
		} else {
			// Convert to absolute path
			if absPath, err := filepath.Abs(request.Directory); err == nil {
				parts = append(parts, absPath)
			} else {
				parts = append(parts, request.Directory)
			}
		}
	}

	return strings.Join(parts, "\n")
}



// buildTemplateData builds the template data context
func (o *Orchestrator) buildTemplateData(request *models.PromptRequest, cfg *interfaces.Config) (*interfaces.TemplateData, error) {
	cwd, _ := os.Getwd()
	
	// Build environment map
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Build config map
	configMap := map[string]interface{}{
		"prompts_location":    cfg.PromptsLocation,
		"editor":              cfg.Editor,
		"default_pre":         cfg.DefaultPre,
		"default_post":        cfg.DefaultPost,
		"fix_file":            cfg.FixFile,
		"max_file_size_bytes": cfg.MaxFileSizeBytes,
		"max_total_bytes":     cfg.MaxTotalBytes,
		"allow_oversize":      cfg.AllowOversize,
		"directory_strategy":  cfg.DirectoryStrategy,
		"target":              cfg.Target,
	}

	// Build git info
	gitInfo := o.buildGitInfo()

	// Build fix info
	fixInfo := interfaces.FixInfo{
		Enabled: request.FixMode,
	}
	if request.FixMode && request.FixFile != "" {
		if content, err := o.loadFixContent(request.FixFile); err == nil {
			fixInfo.Raw = content
			// Try to parse command and output (simple implementation)
			lines := strings.Split(content, "\n")
			if len(lines) > 0 {
				fixInfo.Command = lines[0]
				if len(lines) > 1 {
					fixInfo.Output = strings.Join(lines[1:], "\n")
				}
			}
		}
	}

	return &interfaces.TemplateData{
		Prompt: request.BasePrompt,
		Now:    time.Now(),
		CWD:    cwd,
		Files:  []interfaces.FileInfo{}, // No longer used
		Git:    gitInfo,
		Config: configMap,
		Env:    envMap,
		Fix:    fixInfo,
	}, nil
}

// buildGitInfo builds git repository information
func (o *Orchestrator) buildGitInfo() interfaces.GitInfo {
	gitInfo := interfaces.GitInfo{}
	
	// This is a simple implementation - in a real scenario we'd use git libraries
	// For now, we'll just try to detect if we're in a git repo
	if _, err := os.Stat(".git"); err == nil {
		if cwd, err := os.Getwd(); err == nil {
			gitInfo.Root = cwd
		}
		// TODO: Implement proper git info extraction
		gitInfo.Branch = "main" // Default
		gitInfo.Commit = "unknown"
		gitInfo.Dirty = false
	}
	
	return gitInfo
}

// loadFixContent loads content from the fix file
func (o *Orchestrator) loadFixContent(fixFile string) (string, error) {
	if fixFile == "" {
		return "", fmt.Errorf("fix file path not specified")
	}

	content, err := os.ReadFile(fixFile)
	if err != nil {
		return "", err // Let the caller wrap this with appropriate error type
	}

	trimmedContent := strings.TrimSpace(string(content))
	if trimmedContent == "" {
		return "", fmt.Errorf("fix file is empty")
	}

	return trimmedContent, nil
}

// OutputPrompt handles the final output of the generated prompt
func (o *Orchestrator) OutputPrompt(prompt string, request *models.PromptRequest, cfg *interfaces.Config) error {
	target := request.Target
	if target == "" {
		target = cfg.Target
	}
	if target == "" {
		target = "stdout" // Default fallback
	}

	// Handle different output targets
	switch {
	case target == "clipboard":
		if err := o.outputHandler.WriteToClipboard(prompt); err != nil {
			outputErr := NewOutputError(target, err)
			// Try to recover by falling back to stdout
			if IsRecoverableError(outputErr) {
				fmt.Fprintf(os.Stderr, "Warning: %s\nFalling back to stdout:\n\n", outputErr.Error())
				return o.outputHandler.WriteToStdout(prompt)
			}
			return RecoverFromError(outputErr)
		}
		fmt.Println("Prompt copied to clipboard")
		
	case target == "stdout":
		if err := o.outputHandler.WriteToStdout(prompt); err != nil {
			outputErr := NewOutputError(target, err)
			return RecoverFromError(outputErr)
		}
		
	case strings.HasPrefix(target, "file:"):
		filePath := strings.TrimPrefix(target, "file:")
		if err := o.outputHandler.WriteToFile(prompt, filePath); err != nil {
			outputErr := NewOutputError(target, err)
			return RecoverFromError(outputErr)
		}
		fmt.Printf("Prompt written to %s\n", filePath)
		
	default:
		return RecoverFromError(NewValidationError("target", target, "unsupported output target"))
	}

	// Handle editor integration if explicitly requested
	if request.EditorRequested {
		editor := o.resolveEditor(request.Editor, cfg.Editor)
		if err := o.outputHandler.OpenInEditor(prompt, editor); err != nil {
			outputErr := NewOutputError("editor", err)
			return RecoverFromError(outputErr)
		}
	}

	return nil
}

// validateRequest validates the prompt request
func (o *Orchestrator) validateRequest(request *models.PromptRequest) error {
	if request == nil {
		return NewValidationError("request", nil, "request cannot be nil")
	}

	// In noninteractive mode, base prompt is required unless in fix mode
	if !request.Interactive && request.BasePrompt == "" && !request.FixMode {
		return NewValidationError("base_prompt", "", "required in noninteractive mode")
	}

	// Validate target format if specified
	if request.Target != "" {
		validTargets := []string{"clipboard", "stdout"}
		isValid := false
		for _, valid := range validTargets {
			if request.Target == valid || strings.HasPrefix(request.Target, "file:") {
				isValid = true
				break
			}
		}
		if !isValid {
			return NewValidationError("target", request.Target, "must be 'clipboard', 'stdout', or 'file:/path'")
		}
	}

	// Validate config path if specified
	if request.ConfigPath != "" {
		if _, err := os.Stat(request.ConfigPath); os.IsNotExist(err) {
			return NewValidationError("config_path", request.ConfigPath, "file does not exist")
		}
	}

	// Validate template names if specified
	if request.PreTemplate != "" && strings.TrimSpace(request.PreTemplate) == "" {
		return NewValidationError("template_name", request.PreTemplate, "pre-template name cannot be empty")
	}
	if request.PostTemplate != "" && strings.TrimSpace(request.PostTemplate) == "" {
		return NewValidationError("template_name", request.PostTemplate, "post-template name cannot be empty")
	}

	return nil
}

// resolveEditor resolves the editor using precedence rules
func (o *Orchestrator) resolveEditor(requestEditor, configEditor string) string {
	// Precedence: --editor flag > $VISUAL > $EDITOR > config editor > nvim > vi
	if requestEditor != "" {
		return requestEditor
	}
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if configEditor != "" {
		return configEditor
	}
	// Try common editors as fallback
	for _, editor := range []string{"nvim", "vim", "vi", "nano"} {
		if _, err := os.Stat("/usr/bin/" + editor); err == nil {
			return editor
		}
	}
	return "vi" // Final fallback
}