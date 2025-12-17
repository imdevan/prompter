package orchestrator

import (
	"errors"
	"strings"
	"testing"

	"prompter-cli/pkg/models"
)

func TestOrchestrator_validateRequest(t *testing.T) {
	orch := New()

	tests := []struct {
		name    string
		request *models.PromptRequest
		wantErr bool
		errType error
	}{
		{
			name:    "nil request",
			request: nil,
			wantErr: true,
			errType: ErrValidationFailed,
		},
		{
			name: "valid interactive request",
			request: &models.PromptRequest{
				Interactive: true,
				BasePrompt:  "",
			},
			wantErr: false,
		},
		{
			name: "invalid noninteractive request without base prompt",
			request: &models.PromptRequest{
				Interactive: false,
				BasePrompt:  "",
				FixMode:     false,
			},
			wantErr: true,
			errType: ErrValidationFailed,
		},
		{
			name: "valid noninteractive fix mode",
			request: &models.PromptRequest{
				Interactive: false,
				BasePrompt:  "",
				FixMode:     true,
			},
			wantErr: false,
		},
		{
			name: "invalid target",
			request: &models.PromptRequest{
				Interactive: false,
				BasePrompt:  "test",
				Target:      "invalid-target",
			},
			wantErr: true,
			errType: ErrValidationFailed,
		},
		{
			name: "valid file target",
			request: &models.PromptRequest{
				Interactive: false,
				BasePrompt:  "test",
				Target:      "file:/tmp/test.txt",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orch.validateRequest(tt.request)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				
				// Check if it's a PrompterError with the right type
				var prompterErr *PrompterError
				if errors.As(err, &prompterErr) {
					if !errors.Is(prompterErr.Type, tt.errType) {
						t.Errorf("Expected error type %v, got %v", tt.errType, prompterErr.Type)
					}
					// Verify error has guidance
					if prompterErr.Guidance == "" {
						t.Errorf("Expected error to have guidance, got empty string")
					}
				} else {
					t.Errorf("Expected PrompterError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPrompterError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *PrompterError
		wantText string
	}{
		{
			name: "error with guidance",
			err: &PrompterError{
				Type:     ErrValidationFailed,
				Message:  "test message",
				Guidance: "test guidance",
			},
			wantText: "validation error: test message\n\nSuggestion: test guidance",
		},
		{
			name: "error without guidance",
			err: &PrompterError{
				Type:    ErrConfigurationInvalid,
				Message: "config error",
			},
			wantText: "configuration error: config error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantText {
				t.Errorf("PrompterError.Error() = %q, want %q", got, tt.wantText)
			}
		})
	}
}

func TestNewConfigurationError(t *testing.T) {
	cause := errors.New("file not found")
	err := NewConfigurationError("config file missing", cause)

	if !errors.Is(err.Type, ErrConfigurationInvalid) {
		t.Errorf("Expected error type %v, got %v", ErrConfigurationInvalid, err.Type)
	}

	if err.Guidance == "" {
		t.Errorf("Expected guidance to be set")
	}

	if !strings.Contains(err.Guidance, "configuration file") {
		t.Errorf("Expected guidance to mention configuration file, got: %s", err.Guidance)
	}

	if !errors.Is(err, cause) {
		t.Errorf("Expected error to wrap cause")
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		recoverable bool
	}{
		{
			name: "template not found error",
			err: &PrompterError{
				Type:    ErrTemplateNotFound,
				Message: "template not found",
			},
			recoverable: true,
		},
		{
			name: "clipboard output error",
			err: &PrompterError{
				Type:    ErrOutputFailed,
				Message: "clipboard failed",
			},
			recoverable: true,
		},
		{
			name: "configuration error",
			err: &PrompterError{
				Type:    ErrConfigurationInvalid,
				Message: "config invalid",
			},
			recoverable: false,
		},
		{
			name:        "non-prompter error",
			err:         errors.New("regular error"),
			recoverable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRecoverableError(tt.err)
			if got != tt.recoverable {
				t.Errorf("IsRecoverableError() = %v, want %v", got, tt.recoverable)
			}
		})
	}
}