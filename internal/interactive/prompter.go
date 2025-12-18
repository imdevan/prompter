package interactive

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
	"github.com/atotto/clipboard"
	"golang.org/x/term"
	"prompter-cli/pkg/models"
)

// Prompter handles interactive user input collection
type Prompter struct {
	promptsLocation string
}

// NewPrompter creates a new interactive prompter
func NewPrompter(promptsLocation string) *Prompter {
	return &Prompter{
		promptsLocation: promptsLocation,
	}
}

// CollectMissingInputs prompts the user for any missing required inputs
func (p *Prompter) CollectMissingInputs(request *models.PromptRequest) error {
	// Handle clipboard reading - append to existing prompt or use as base prompt
	// This should work in both interactive and non-interactive modes
	if request.FromClipboard && !request.FixMode {
		if err := p.appendClipboardToPrompt(request); err != nil {
			return fmt.Errorf("failed to read from clipboard: %w", err)
		}
	}

	if !request.Interactive {
		return nil // Skip interactive prompts in noninteractive mode
	}

	// Collect base prompt if missing and not in fix mode (only in interactive mode)
	if request.BasePrompt == "" && !request.FixMode && request.Interactive {
		if err := p.promptForBasePrompt(request); err != nil {
			return fmt.Errorf("failed to collect base prompt: %w", err)
		}
	}

	// Collect pre-template if not specified
	if request.PreTemplate == "" && !request.FixMode {
		if err := p.promptForPreTemplate(request); err != nil {
			return fmt.Errorf("failed to collect pre-template: %w", err)
		}
	}

	// Collect post-template if not specified
	if request.PostTemplate == "" && !request.FixMode {
		if err := p.promptForPostTemplate(request); err != nil {
			return fmt.Errorf("failed to collect post-template: %w", err)
		}
	}

	// Collect directory inclusion if not specified
	if request.Directory == "" && len(request.Files) == 0 && !request.FixMode {
		if err := p.promptForDirectoryInclusion(request); err != nil {
			return fmt.Errorf("failed to collect directory inclusion: %w", err)
		}
	}

	// Show confirmation summary
	if err := p.showConfirmationSummary(request); err != nil {
		return fmt.Errorf("user cancelled operation: %w", err)
	}

	return nil
}

// appendClipboardToPrompt reads from clipboard and appends to existing prompt or uses as base prompt
func (p *Prompter) appendClipboardToPrompt(request *models.PromptRequest) error {
	clipboardContent, err := clipboard.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read from clipboard: %w", err)
	}

	clipboardContent = strings.TrimSpace(clipboardContent)
	if clipboardContent == "" {
		return fmt.Errorf("clipboard is empty")
	}

	if request.BasePrompt == "" {
		// No existing prompt, use clipboard content as base prompt
		request.BasePrompt = clipboardContent
		if request.Interactive {
			fmt.Printf("Read base prompt from clipboard: %s\n", truncateString(clipboardContent, 100))
		}
	} else {
		// Append clipboard content to existing prompt
		request.BasePrompt = request.BasePrompt + "\n\n" + clipboardContent
		if request.Interactive {
			fmt.Printf("Appended clipboard content to base prompt: %s\n", truncateString(clipboardContent, 100))
		}
	}
	
	return nil
}

// promptForBasePrompt asks the user to enter a base prompt
func (p *Prompter) promptForBasePrompt(request *models.PromptRequest) error {
	prompt := &survey.Input{
		Message: "Enter your base prompt:",
		Help:    "This is the main prompt text that will be sent to the AI",
	}

	var basePrompt string
	if err := survey.AskOne(prompt, &basePrompt, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	request.BasePrompt = strings.TrimSpace(basePrompt)
	return nil
}

// promptForPreTemplate asks the user to select a pre-template
func (p *Prompter) promptForPreTemplate(request *models.PromptRequest) error {
	templates, err := p.findTemplates("pre")
	if err != nil {
		return fmt.Errorf("failed to find pre templates: %w", err)
	}

	// Build options with proper ordering: defaults first, then "None", then regulars
	options := p.buildOptionsWithNone(templates, "pre")

	selected, err := p.selectTemplate(options, "Select a pre-template (prepended to prompt):", "Pre-templates are added before your base prompt", request.NumberSelect)
	if err != nil {
		return err
	}

	if selected != "None" {
		request.PreTemplate = selected
	}

	return nil
}

// promptForPostTemplate asks the user to select a post-template
func (p *Prompter) promptForPostTemplate(request *models.PromptRequest) error {
	templates, err := p.findTemplates("post")
	if err != nil {
		return fmt.Errorf("failed to find post templates: %w", err)
	}

	// Build options with proper ordering: defaults first, then "None", then regulars
	options := p.buildOptionsWithNone(templates, "post")

	selected, err := p.selectTemplate(options, "Select a post-template (appended to prompt):", "Post-templates are added after your base prompt", request.NumberSelect)
	if err != nil {
		return err
	}

	if selected != "None" {
		request.PostTemplate = selected
	}

	return nil
}

// promptForDirectoryInclusion asks whether to include directory context
func (p *Prompter) promptForDirectoryInclusion(request *models.PromptRequest) error {
	includeDirectory, err := p.selectYesNo(
		"Include current directory context in the prompt?",
		"This will include relevant files from the current directory",
		false, // default to No
		request.NumberSelect,
	)
	if err != nil {
		return err
	}

	if includeDirectory {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		request.Directory = cwd
	}

	return nil
}

// showConfirmationSummary displays a summary and asks for confirmation
func (p *Prompter) showConfirmationSummary(request *models.PromptRequest) error {
	// Skip confirmation prompt entirely
	return nil
}

// findTemplates discovers available templates in the specified subdirectory
func (p *Prompter) findTemplates(subdir string) ([]string, error) {
	templateDir := filepath.Join(p.promptsLocation, subdir)
	
	// Check if directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if directory doesn't exist
	}

	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read template directory %s: %w", templateDir, err)
	}

	var defaultTemplates []string
	var regularTemplates []string
	
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			// Remove .md extension for processing
			name := strings.TrimSuffix(entry.Name(), ".md")
			
			// Check if this is a default template
			if strings.Contains(name, ".default.") {
				// Strip the .default. part for display
				displayName := strings.ReplaceAll(name, ".default.", ".")
				// Remove any leading/trailing dots that might result
				displayName = strings.Trim(displayName, ".")
				defaultTemplates = append(defaultTemplates, displayName)
			} else if strings.HasSuffix(name, ".default") {
				// Handle case where .default is at the end
				displayName := strings.TrimSuffix(name, ".default")
				defaultTemplates = append(defaultTemplates, displayName)
			} else {
				regularTemplates = append(regularTemplates, name)
			}
		}
	}

	// Combine lists with defaults first
	var templates []string
	templates = append(templates, defaultTemplates...)
	templates = append(templates, regularTemplates...)

	return templates, nil
}



// buildOptionsWithNone constructs the options list with proper ordering:
// default templates first, then "None", then regular templates
func (p *Prompter) buildOptionsWithNone(templates []string, subdir string) []string {
	// We need to separate default templates from regular templates
	// to insert "None" in the right place
	templateDir := filepath.Join(p.promptsLocation, subdir)
	
	var defaultTemplates []string
	var regularTemplates []string
	
	// Check if directory exists
	if entries, err := os.ReadDir(templateDir); err == nil {
		// Build a map of which templates are defaults
		defaultNames := make(map[string]bool)
		
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				name := strings.TrimSuffix(entry.Name(), ".md")
				
				// Check if this is a default template
				if strings.Contains(name, ".default.") || strings.HasSuffix(name, ".default") {
					var displayName string
					if strings.Contains(name, ".default.") {
						displayName = strings.ReplaceAll(name, ".default.", ".")
						displayName = strings.Trim(displayName, ".")
					} else {
						displayName = strings.TrimSuffix(name, ".default")
					}
					defaultNames[displayName] = true
				}
			}
		}
		
		// Separate templates based on whether they're defaults
		for _, template := range templates {
			if defaultNames[template] {
				defaultTemplates = append(defaultTemplates, template)
			} else {
				regularTemplates = append(regularTemplates, template)
			}
		}
	} else {
		// Fallback: if we can't read the directory, treat all as regular
		regularTemplates = templates
	}
	
	// Build final options list: defaults first, then "None", then regulars
	var options []string
	options = append(options, defaultTemplates...)
	options = append(options, "None")
	options = append(options, regularTemplates...)
	
	return options
}

// selectTemplate handles template selection with optional number key support
func (p *Prompter) selectTemplate(options []string, message, help string, numberSelect bool) (string, error) {
	if len(options) == 0 {
		return "None", nil
	}

	if numberSelect {
		return p.selectTemplateWithNumbers(options, message, help)
	}

	// Use regular survey selection
	prompt := &survey.Select{
		Message: message,
		Options: options,
		Help:    help,
	}

	var selected string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", err
	}

	return selected, nil
}

// selectTemplateWithNumbers displays numbered options and allows instant selection by number key
func (p *Prompter) selectTemplateWithNumbers(options []string, message, help string) (string, error) {
	fmt.Printf("\n%s\n", message)
	if help != "" {
		fmt.Printf("  %s (Press number key for instant selection or use arrow keys)\n", help)
	}
	fmt.Println()

	// Display numbered options
	for i, option := range options {
		fmt.Printf("  %d. %s\n", i+1, option)
	}
	fmt.Println()

	// Check if we're in a terminal that supports raw mode
	if !term.IsTerminal(int(syscall.Stdin)) {
		// Fallback to regular input if not in a terminal
		return p.fallbackNumberSelection(options)
	}

	// Save the current terminal state
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		// Fallback to regular input if raw mode fails
		return p.fallbackNumberSelection(options)
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	fmt.Print("Select option: ")

	// Read single character input
	buffer := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(buffer)
		if err != nil {
			return "", err
		}

		char := buffer[0]

		// Handle number keys (1-9)
		if char >= '1' && char <= '9' {
			selectedIndex := int(char - '1') // Convert '1' to 0, '2' to 1, etc.
			if selectedIndex < len(options) {
				fmt.Printf("%c\n", char) // Echo the pressed key
				return options[selectedIndex], nil
			}
		}

		// Handle Enter key (fallback to first option or None)
		if char == '\r' || char == '\n' {
			fmt.Println()
			if len(options) > 0 {
				return options[0], nil
			}
			return "None", nil
		}

		// Handle Escape or Ctrl+C
		if char == 27 || char == 3 {
			fmt.Println()
			return "", fmt.Errorf("selection cancelled")
		}

		// For any other key, continue waiting
	}
}

// fallbackNumberSelection provides a fallback when raw terminal mode is not available
func (p *Prompter) fallbackNumberSelection(options []string) (string, error) {
	fmt.Printf("Enter number (1-%d) or press Enter for first option: ", len(options))
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		if len(options) > 0 {
			return options[0], nil
		}
		return "None", nil
	}

	// Try to parse as number
	selectedIndex, err := strconv.Atoi(input)
	if err != nil {
		return "", fmt.Errorf("invalid input: please enter a number between 1 and %d", len(options))
	}

	// Validate range (convert from 1-based to 0-based)
	if selectedIndex < 1 || selectedIndex > len(options) {
		return "", fmt.Errorf("invalid selection: please enter a number between 1 and %d", len(options))
	}

	return options[selectedIndex-1], nil
}

// selectYesNo handles yes/no selection with optional number key support
func (p *Prompter) selectYesNo(message, help string, defaultValue, numberSelect bool) (bool, error) {
	if numberSelect {
		return p.selectYesNoWithNumbers(message, help, defaultValue)
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
func (p *Prompter) selectYesNoWithNumbers(message, help string, defaultValue bool) (bool, error) {
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
		return p.fallbackYesNoSelection(defaultValue)
	}

	// Save the current terminal state
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		// Fallback to regular input if raw mode fails
		return p.fallbackYesNoSelection(defaultValue)
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
func (p *Prompter) fallbackYesNoSelection(defaultValue bool) (bool, error) {
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

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
// CollectTemplateInfo asks the user for template type and name
func (p *Prompter) CollectTemplateInfo() (string, string, error) {
	// Ask for template type
	templateTypePrompt := &survey.Select{
		Message: "Select template type:",
		Options: []string{"pre", "post"},
		Help:    "Pre-templates are added before your prompt, post-templates are added after",
	}

	var templateType string
	if err := survey.AskOne(templateTypePrompt, &templateType); err != nil {
		return "", "", err
	}

	// Ask for template name
	namePrompt := &survey.Input{
		Message: "Enter template name:",
		Help:    "This will be the filename (without .md extension)",
	}

	var templateName string
	if err := survey.AskOne(namePrompt, &templateName, survey.WithValidator(survey.Required)); err != nil {
		return "", "", err
	}

	// Clean the template name (remove any .md extension if user added it)
	templateName = strings.TrimSuffix(templateName, ".md")
	templateName = strings.TrimSpace(templateName)

	return templateType, templateName, nil
}

// CollectTemplateContent asks the user for template content
func (p *Prompter) CollectTemplateContent() (string, error) {
	contentPrompt := &survey.Multiline{
		Message: "Enter template content:",
		Help:    "Enter the template content. Press Ctrl+D (Unix) or Ctrl+Z (Windows) when finished",
	}

	var content string
	if err := survey.AskOne(contentPrompt, &content, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return strings.TrimSpace(content), nil
}

// ConfirmOverwrite asks the user if they want to overwrite an existing file
func (p *Prompter) ConfirmOverwrite(filePath string) (bool, error) {
	overwritePrompt := &survey.Confirm{
		Message: fmt.Sprintf("Template file already exists: %s. Overwrite?", filePath),
		Default: false,
	}

	var overwrite bool
	if err := survey.AskOne(overwritePrompt, &overwrite); err != nil {
		return false, err
	}

	return overwrite, nil
}