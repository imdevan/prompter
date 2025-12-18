package orchestrator

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"golang.org/x/term"
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
		outputHandler:     NewOutputHandler(),
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

// GetTemplateProcessor returns the template processor (exported for app layer)
func (o *Orchestrator) GetTemplateProcessor() interfaces.TemplateProcessor {
	return o.templateProcessor
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

	// Update template processor with the loaded configuration
	if processor, ok := o.templateProcessor.(*template.Processor); ok {
		processor.SetPromptsLocation(cfg.PromptsLocation)
		processor.SetLocalPromptsFromConfig(cfg.LocalPromptsLocation)
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
	// In fix mode, don't set fix file from config - let it read from stdin if not explicitly set
	if !request.FixMode && request.FixFile == "" && cfg.FixFile != "" {
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
	// Load fix content from file, re-run command, or stdin
	fixContent, err := o.loadFixContent(request.FixFile, request.Interactive, request.NumberSelect)
	if err != nil {
		fixErr := NewFixModeError(request.FixFile, err)
		return "", RecoverFromError(fixErr)
	}

	var promptParts []string

	// Try to load fix.md from prompts_location root, fallback to "Please fix"
	fixPrompt, err := o.loadFixPrompt(cfg.PromptsLocation)
	if err != nil {
		// Fallback to default "Please fix" prompt
		fixPrompt = "Please fix"
	}
	
	// Add the fix prompt
	promptParts = append(promptParts, fixPrompt)

	// Add the captured content (command + output) as a separate part
	promptParts = append(promptParts, fixContent)

	return strings.Join(promptParts, "\n\n"), nil
}

// processTemplate processes a template with the current context
func (o *Orchestrator) processTemplate(templateName string, request *models.PromptRequest, cfg *interfaces.Config, templateType string) (string, error) {
	// Update template processor with prompts location
	if processor, ok := o.templateProcessor.(*template.Processor); ok {
		processor.SetPromptsLocation(cfg.PromptsLocation)
		processor.SetLocalPromptsFromConfig(cfg.LocalPromptsLocation)
	}

	// Load template using the template processor's discovery mechanism
	// The processor will find the correct file (including .default. files)
	tmpl, err := o.templateProcessor.LoadTemplate(templateName)
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
		"prompts_location":   cfg.PromptsLocation,
		"editor":             cfg.Editor,
		"default_pre":        cfg.DefaultPre,
		"default_post":       cfg.DefaultPost,
		"fix_file":           cfg.FixFile,
		"directory_strategy": cfg.DirectoryStrategy,
		"target":             cfg.Target,
	}

	// Build git info
	gitInfo := o.buildGitInfo()

	// Build fix info
	fixInfo := interfaces.FixInfo{
		Enabled: request.FixMode,
	}
	if request.FixMode && request.FixFile != "" {
		if content, err := o.loadFixContent(request.FixFile, request.Interactive, request.NumberSelect); err == nil {
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

// loadFixContent loads content from the fix file, re-runs last command, or reads from stdin
func (o *Orchestrator) loadFixContent(fixFile string, interactive bool, numberSelect bool) (string, error) {
	if fixFile != "" {
		// Read from specified file
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

	// No fix file specified - try to re-run the last command
	if interactive {
		// Interactive mode: prompt user to re-run last command
		return o.promptAndRerunLastCommand(numberSelect)
	} else {
		// Non-interactive mode: automatically re-run last command
		return o.rerunLastCommand()
	}
}

// readFromStdin reads all content from stdin
func (o *Orchestrator) readFromStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

// captureTerminalOutput attempts to capture previous terminal output
func (o *Orchestrator) captureTerminalOutput() (string, error) {
	// Check if we can access terminal scroll buffer (advanced terminals)
	if content, err := o.tryAdvancedTerminalCapture(); err == nil && content != "" {
		return content, nil
	}

	// Try to get recent shell history as fallback
	if content, err := o.tryShellHistory(); err == nil && content != "" {
		return content, nil
	}

	// Provide helpful error message
	return "", fmt.Errorf(`unable to capture terminal output automatically.

For best results, use one of these methods:
1. Pipe command output: command 2>&1 | ./prompter --fix --yes
2. Save to file first: command 2>&1 | tee /tmp/output.txt && ./prompter --fix --yes
3. Use terminal session recording tools like 'script' or 'asciinema'

Current fallback captured recent shell history only.`)
}

// tryAdvancedTerminalCapture attempts advanced terminal output capture
func (o *Orchestrator) tryAdvancedTerminalCapture() (string, error) {
	// This would require terminal-specific implementations
	// For now, return error to fall back to shell history
	return "", fmt.Errorf("advanced terminal capture not available")
}

// tryShellHistory attempts to get recent commands and their context
func (o *Orchestrator) tryShellHistory() (string, error) {
	// Try to read recent shell history
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Check for zsh history
	historyFile := filepath.Join(homeDir, ".zsh_history")
	if _, err := os.Stat(historyFile); err == nil {
		return o.readRecentHistory(historyFile, "zsh")
	}

	// Check for bash history
	historyFile = filepath.Join(homeDir, ".bash_history")
	if _, err := os.Stat(historyFile); err == nil {
		return o.readRecentHistory(historyFile, "bash")
	}

	return "", fmt.Errorf("no shell history found")
}

// readRecentHistory reads recent commands from shell history
func (o *Orchestrator) readRecentHistory(historyFile, shell string) (string, error) {
	content, err := os.ReadFile(historyFile)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("insufficient history")
	}

	// Get the last few commands, excluding the current prompter command
	var recentLines []string

	// Work backwards through history to find recent commands
	for i := len(lines) - 1; i >= 0 && len(recentLines) < 5; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// For zsh, remove timestamp if present
		if shell == "zsh" && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) == 2 {
				line = parts[1]
			}
		}

		// Skip the current prompter command to avoid recursion
		if strings.Contains(line, "prompter") && strings.Contains(line, "--fix") {
			continue
		}

		recentLines = append([]string{"$ " + line}, recentLines...)
	}

	if len(recentLines) == 0 {
		return "", fmt.Errorf("no recent commands found")
	}

	// Add a note about output capture limitation
	result := strings.Join(recentLines, "\n")
	result += "\n\n# Note: Command output not captured. For full output capture, use: command 2>&1 | tee /tmp/output.txt && ./prompter --fix --yes"

	return result, nil
}

// promptAndRerunLastCommand prompts user to re-run the last command and captures output
func (o *Orchestrator) promptAndRerunLastCommand(numberSelect bool) (string, error) {
	// Get the last command from history
	lastCmd, err := o.getLastCommand()
	if err != nil {
		return "", fmt.Errorf("failed to get last command: %w", err)
	}

	// Prompt user to confirm re-running the command
	confirmed, err := o.selectYesNo(
		fmt.Sprintf("Re-run last command to capture output?\n  $ %s", lastCmd),
		"This will execute the command and capture its output for fixing",
		true, // default to Yes
		numberSelect,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get user confirmation: %w", err)
	}

	if !confirmed {
		return "", fmt.Errorf("user declined to re-run command")
	}

	// Execute the command and capture output
	return o.executeAndCaptureCommand(lastCmd)
}

// rerunLastCommand automatically re-runs the last command (non-interactive mode)
func (o *Orchestrator) rerunLastCommand() (string, error) {
	// Get the last command from history
	lastCmd, err := o.getLastCommand()
	if err != nil {
		return "", fmt.Errorf("failed to get last command: %w", err)
	}

	fmt.Printf("Re-running last command: %s\n", lastCmd)

	// Execute the command and capture output
	return o.executeAndCaptureCommand(lastCmd)
}

// getLastCommand retrieves the last command from shell history
func (o *Orchestrator) getLastCommand() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Check for zsh history first
	historyFile := filepath.Join(homeDir, ".zsh_history")
	if _, err := os.Stat(historyFile); err == nil {
		return o.getLastCommandFromHistory(historyFile, "zsh")
	}

	// Check for bash history
	historyFile = filepath.Join(homeDir, ".bash_history")
	if _, err := os.Stat(historyFile); err == nil {
		return o.getLastCommandFromHistory(historyFile, "bash")
	}

	return "", fmt.Errorf("no shell history found")
}

// getLastCommandFromHistory extracts the last command from a history file
func (o *Orchestrator) getLastCommandFromHistory(historyFile, shell string) (string, error) {
	content, err := os.ReadFile(historyFile)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")

	// Work backwards to find the last non-prompter command
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// For zsh, remove timestamp if present
		if shell == "zsh" && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) == 2 {
				line = parts[1]
			}
		}

		// Skip prompter commands to avoid recursion
		if strings.Contains(line, "prompter") {
			continue
		}

		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			return line, nil
		}
	}

	return "", fmt.Errorf("no suitable command found in history")
}

// executeAndCaptureCommand executes a command and captures both stdout and stderr
func (o *Orchestrator) executeAndCaptureCommand(command string) (string, error) {
	// Execute the command using the shell
	cmd := exec.Command("sh", "-c", command)

	// Capture both stdout and stderr
	output, _ := cmd.CombinedOutput()

	// Format the result with command and output separated by a blank line
	var result strings.Builder
	result.WriteString("$ ")
	result.WriteString(command)
	result.WriteString("\n\n")
	result.Write(output)

	return strings.TrimSpace(result.String()), nil
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

	// In noninteractive mode, base prompt is required unless in fix mode or clipboard flag is used
	if !request.Interactive && request.BasePrompt == "" && !request.FixMode && !request.FromClipboard {
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

// selectYesNo handles yes/no selection with optional number key support
func (o *Orchestrator) selectYesNo(message, help string, defaultValue, numberSelect bool) (bool, error) {
	if numberSelect {
		return o.selectYesNoWithNumbers(message, help, defaultValue)
	}

	// Use regular survey confirm
	prompt := &survey.Confirm{
		Message: message,
		Help:    help,
		Default: defaultValue,
	}

	var result bool
	if err := survey.AskOne(prompt, &result); err != nil {
		return false, err
	}

	return result, nil
}

// selectYesNoWithNumbers displays numbered yes/no options and allows instant selection
func (o *Orchestrator) selectYesNoWithNumbers(message, help string, defaultValue bool) (bool, error) {
	fmt.Printf("\n%s\n", message)
	if help != "" {
		fmt.Printf("  %s (Press number key for instant selection)\n", help)
	}
	fmt.Println()

	// Display options with default marked
	if defaultValue {
		fmt.Println("  1. Yes (default)")
		fmt.Println("  2. No")
	} else {
		fmt.Println("  1. Yes")
		fmt.Println("  2. No (default)")
	}
	fmt.Println()

	// Check if we're in a terminal that supports raw mode
	if !term.IsTerminal(int(syscall.Stdin)) {
		// Fallback to regular input if not in a terminal
		return o.fallbackYesNoSelection(defaultValue)
	}

	// Save the current terminal state
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		// Fallback to regular input if raw mode fails
		return o.fallbackYesNoSelection(defaultValue)
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	fmt.Print("Select option: ")

	// Read single character input
	buffer := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(buffer)
		if err != nil {
			return false, err
		}

		char := buffer[0]

		// Handle number keys
		if char == '1' {
			fmt.Printf("1\n")
			return true, nil // Yes
		}
		if char == '2' {
			fmt.Printf("2\n")
			return false, nil // No
		}

		// Handle Enter key (use default)
		if char == '\r' || char == '\n' {
			fmt.Println()
			return defaultValue, nil
		}

		// Handle Escape or Ctrl+C
		if char == 27 || char == 3 {
			fmt.Println()
			return false, fmt.Errorf("selection cancelled")
		}

		// For any other key, continue waiting
	}
}

// fallbackYesNoSelection provides a fallback when raw terminal mode is not available
func (o *Orchestrator) fallbackYesNoSelection(defaultValue bool) (bool, error) {
	defaultText := "No"
	if defaultValue {
		defaultText = "Yes"
	}

	fmt.Printf("Enter 1 for Yes, 2 for No, or press Enter for default (%s): ", defaultText)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}

	switch input {
	case "1":
		return true, nil
	case "2":
		return false, nil
	default:
		return false, fmt.Errorf("invalid input: please enter 1 for Yes or 2 for No")
	}
}

// loadFixPrompt loads the fix prompt from prompts_location/fix.md
func (o *Orchestrator) loadFixPrompt(promptsLocation string) (string, error) {
	fixPath := filepath.Join(promptsLocation, "fix.md")
	
	content, err := os.ReadFile(fixPath)
	if err != nil {
		return "", fmt.Errorf("fix.md not found at %s: %w", fixPath, err)
	}
	
	return strings.TrimSpace(string(content)), nil
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

