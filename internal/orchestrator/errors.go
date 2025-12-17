package orchestrator

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Error types for different categories of failures
var (
	ErrConfigurationInvalid = errors.New("configuration error")
	ErrTemplateNotFound     = errors.New("template not found")
	ErrTemplateInvalid      = errors.New("template error")
	ErrContentCollection    = errors.New("content collection error")
	ErrFixModeInvalid       = errors.New("fix mode error")
	ErrOutputFailed         = errors.New("output error")
	ErrValidationFailed     = errors.New("validation error")
)

// PrompterError represents a structured error with actionable guidance
type PrompterError struct {
	Type     error
	Message  string
	Guidance string
	Cause    error
}

func (e *PrompterError) Error() string {
	if e.Guidance != "" {
		return fmt.Sprintf("%s: %s\n\nSuggestion: %s", e.Type, e.Message, e.Guidance)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *PrompterError) Unwrap() error {
	return e.Cause
}

// Error constructors with actionable guidance

func NewConfigurationError(message string, cause error) *PrompterError {
	guidance := "Check your configuration file syntax and ensure all paths exist. " +
		"Use 'prompter --config /path/to/config.toml' to specify a different config file."
	
	if strings.Contains(message, "permission") {
		guidance = "Check file permissions for your configuration directory. " +
			"Ensure you have read/write access to ~/.config/prompter/"
	} else if strings.Contains(message, "not found") || strings.Contains(message, "does not exist") {
		guidance = "The configuration file doesn't exist. Create ~/.config/prompter/config.toml " +
			"or specify a different path with --config flag."
	}
	
	return &PrompterError{
		Type:     ErrConfigurationInvalid,
		Message:  message,
		Guidance: guidance,
		Cause:    cause,
	}
}

func NewTemplateError(templateName string, cause error) *PrompterError {
	message := fmt.Sprintf("failed to process template '%s'", templateName)
	guidance := fmt.Sprintf("Ensure the template '%s.md' exists in prompts/pre/ or prompts/post/ directory. " +
		"Check template syntax for valid Go template format with {{ }} delimiters.", templateName)
	
	if strings.Contains(cause.Error(), "not found") {
		guidance = fmt.Sprintf("Template '%s' not found. Available templates can be listed by checking " +
			"the prompts/pre/ and prompts/post/ directories. Template names are case-insensitive.", templateName)
	} else if strings.Contains(cause.Error(), "parse") || strings.Contains(cause.Error(), "syntax") {
		guidance = fmt.Sprintf("Template '%s' has syntax errors. Check for proper {{ }} delimiters " +
			"and valid Go template syntax. Ensure all variables are properly referenced.", templateName)
	}
	
	return &PrompterError{
		Type:     ErrTemplateInvalid,
		Message:  message,
		Guidance: guidance,
		Cause:    cause,
	}
}

func NewContentCollectionError(path string, cause error) *PrompterError {
	message := fmt.Sprintf("failed to collect content from '%s'", path)
	guidance := "Ensure the file or directory exists and you have read permissions. " +
		"Check that the path is correct and accessible."
	
	if strings.Contains(cause.Error(), "permission") {
		guidance = fmt.Sprintf("Permission denied accessing '%s'. Ensure you have read permissions " +
			"for the file/directory and all parent directories.", path)
	} else if strings.Contains(cause.Error(), "not found") || strings.Contains(cause.Error(), "does not exist") {
		guidance = fmt.Sprintf("Path '%s' does not exist. Check the path spelling and ensure " +
			"the file or directory exists.", path)
	} else if strings.Contains(cause.Error(), "too large") {
		guidance = fmt.Sprintf("Content from '%s' exceeds size limits. Consider using --allow-oversize " +
			"or increase max_file_size_bytes/max_total_bytes in configuration.", path)
	}
	
	return &PrompterError{
		Type:     ErrContentCollection,
		Message:  message,
		Guidance: guidance,
		Cause:    cause,
	}
}

func NewFixModeError(fixFile string, cause error) *PrompterError {
	message := fmt.Sprintf("fix mode failed with file '%s'", fixFile)
	guidance := fmt.Sprintf("Ensure the fix file '%s' exists and contains captured command output. " +
		"Capture output using: command 2>&1 | tee %s", fixFile, fixFile)
	
	if strings.Contains(cause.Error(), "not found") || strings.Contains(cause.Error(), "does not exist") {
		guidance = fmt.Sprintf("Fix file '%s' does not exist. To use fix mode:\n" +
			"1. Run your failing command: your-command 2>&1 | tee %s\n" +
			"2. Then run: prompter --fix", fixFile, fixFile)
	} else if strings.Contains(cause.Error(), "empty") {
		guidance = fmt.Sprintf("Fix file '%s' is empty. Ensure you captured the command output properly.", fixFile)
	}
	
	return &PrompterError{
		Type:     ErrFixModeInvalid,
		Message:  message,
		Guidance: guidance,
		Cause:    cause,
	}
}

func NewOutputError(target string, cause error) *PrompterError {
	message := fmt.Sprintf("failed to output to target '%s'", target)
	guidance := "Check that the output target is valid and accessible."
	
	if target == "clipboard" {
		guidance = "Clipboard access failed. Ensure you're running in a graphical environment " +
			"or try using --target stdout instead."
	} else if strings.HasPrefix(target, "file:") {
		filePath := strings.TrimPrefix(target, "file:")
		guidance = fmt.Sprintf("Failed to write to file '%s'. Check that the directory exists " +
			"and you have write permissions.", filePath)
	} else if strings.Contains(cause.Error(), "editor") {
		guidance = "Editor launch failed. Check that the specified editor is installed and in PATH. " +
			"Try setting EDITOR environment variable or using --editor flag."
	}
	
	return &PrompterError{
		Type:     ErrOutputFailed,
		Message:  message,
		Guidance: guidance,
		Cause:    cause,
	}
}

func NewValidationError(field string, value interface{}, reason string) *PrompterError {
	message := fmt.Sprintf("validation failed for %s: %v (%s)", field, value, reason)
	guidance := "Check the input value and ensure it meets the required format."
	
	switch field {
	case "base_prompt":
		guidance = "Base prompt is required in non-interactive mode. Provide a prompt as argument " +
			"or remove --yes flag to use interactive mode."
	case "target":
		guidance = "Target must be 'clipboard', 'stdout', or 'file:/path/to/file'. " +
			"Example: --target file:/tmp/prompt.txt"
	case "config_path":
		guidance = "Configuration file path must be valid and accessible. " +
			"Ensure the file exists and you have read permissions."
	case "template_name":
		guidance = "Template name must not be empty and should correspond to a .md file " +
			"in prompts/pre/ or prompts/post/ directory."
	}
	
	return &PrompterError{
		Type:     ErrValidationFailed,
		Message:  message,
		Guidance: guidance,
		Cause:    nil,
	}
}

// Recovery strategies

// RecoverFromError attempts to recover from common errors with fallback strategies
func RecoverFromError(err error) error {
	if err == nil {
		return nil
	}
	
	var prompterErr *PrompterError
	if !errors.As(err, &prompterErr) {
		// Wrap unknown errors
		return &PrompterError{
			Type:     errors.New("unknown error"),
			Message:  err.Error(),
			Guidance: "An unexpected error occurred. Please check your inputs and try again.",
			Cause:    err,
		}
	}
	
	// Apply recovery strategies based on error type
	switch prompterErr.Type {
	case ErrConfigurationInvalid:
		return recoverFromConfigError(prompterErr)
	case ErrTemplateNotFound:
		return recoverFromTemplateError(prompterErr)
	case ErrOutputFailed:
		return recoverFromOutputError(prompterErr)
	default:
		return prompterErr
	}
}

func recoverFromConfigError(err *PrompterError) error {
	// Try to create default config directory if it doesn't exist
	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return err // Can't recover
	}
	
	configDir := fmt.Sprintf("%s/.config/prompter", homeDir)
	if _, statErr := os.Stat(configDir); os.IsNotExist(statErr) {
		if mkdirErr := os.MkdirAll(configDir, 0755); mkdirErr != nil {
			// Add recovery attempt info to guidance
			err.Guidance += fmt.Sprintf("\n\nAttempted to create config directory '%s' but failed: %v", 
				configDir, mkdirErr)
			return err
		}
		
		// Successfully created directory
		err.Guidance += fmt.Sprintf("\n\nCreated config directory '%s'. You can now create a config.toml file there.", 
			configDir)
	}
	
	return err
}

func recoverFromTemplateError(err *PrompterError) error {
	// For template not found errors, we can suggest continuing without the template
	if strings.Contains(err.Message, "not found") {
		err.Guidance += "\n\nYou can continue without this template by omitting the --pre or --post flag."
	}
	return err
}

func recoverFromOutputError(err *PrompterError) error {
	// For clipboard errors, suggest stdout fallback
	if strings.Contains(err.Message, "clipboard") {
		err.Guidance += "\n\nTry using --target stdout as a fallback."
	}
	return err
}

// IsRecoverableError checks if an error can be recovered from
func IsRecoverableError(err error) bool {
	var prompterErr *PrompterError
	if !errors.As(err, &prompterErr) {
		return false
	}
	
	// Some errors are recoverable with user intervention
	switch prompterErr.Type {
	case ErrTemplateNotFound:
		return true // Can continue without template
	case ErrOutputFailed:
		return strings.Contains(prompterErr.Message, "clipboard") // Can fallback to stdout
	default:
		return false
	}
}