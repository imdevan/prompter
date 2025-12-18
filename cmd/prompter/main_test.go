package main

import (
	"testing"

	"github.com/spf13/cobra"
	"prompter-cli/pkg/models"
)

func TestBuildRequestFromFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		flags    map[string]string
		boolFlags map[string]bool
		expected *models.PromptRequest
		wantErr  bool
	}{
		{
			name: "basic request with base prompt",
			args: []string{"test prompt"},
			flags: map[string]string{
				"pre":  "test-pre",
				"post": "test-post",
			},
			expected: &models.PromptRequest{
				BasePrompt:   "test prompt",
				PreTemplate:  "test-pre",
				PostTemplate: "test-post",
				Interactive:  true,
				Files:        []string{},
			},
		},
		{
			name: "noninteractive mode",
			args: []string{"test prompt"},
			boolFlags: map[string]bool{
				"yes": true,
			},
			expected: &models.PromptRequest{
				BasePrompt:          "test prompt",
				Interactive:         true, // Will be resolved later based on config
				ForceNonInteractive: true,
				Files:               []string{},
			},
		},
		{
			name: "fix mode",
			boolFlags: map[string]bool{
				"fix": true,
				"yes": true,
			},
			expected: &models.PromptRequest{
				FixMode:             true,
				Interactive:         true, // Will be resolved later based on config
				ForceNonInteractive: true,
				Files:               []string{},
			},
		},
		{
			name: "number selection mode",
			args: []string{"test prompt"},
			boolFlags: map[string]bool{
				"numbers": true,
			},
			expected: &models.PromptRequest{
				BasePrompt:   "test prompt",
				Interactive:  true,
				NumberSelect: true,
				Files:        []string{},
			},
		},
		{
			name: "clipboard mode",
			boolFlags: map[string]bool{
				"clipboard": true,
			},
			expected: &models.PromptRequest{
				Interactive:   true,
				FromClipboard: true,
				Files:         []string{},
			},
		},
		{
			name: "clipboard with base prompt",
			args: []string{"test prompt"},
			boolFlags: map[string]bool{
				"clipboard": true,
			},
			expected: &models.PromptRequest{
				BasePrompt:    "test prompt",
				Interactive:   true,
				FromClipboard: true,
				Files:         []string{},
			},
		},
		{
			name: "force interactive mode",
			args: []string{"test prompt"},
			boolFlags: map[string]bool{
				"interactive": true,
			},
			expected: &models.PromptRequest{
				BasePrompt:       "test prompt",
				Interactive:      true,
				ForceInteractive: true,
				Files:            []string{},
			},
		},
		{
			name: "conflicting interactive flags should error",
			boolFlags: map[string]bool{
				"interactive": true,
				"yes":         true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			
			// Add flags to command
			cmd.Flags().String("config", "", "")
			cmd.Flags().Bool("yes", false, "")
			cmd.Flags().String("pre", "", "")
			cmd.Flags().String("post", "", "")
			cmd.Flags().StringSlice("file", []string{}, "")
			cmd.Flags().BoolP("directory", "d", false, "")
			cmd.Flags().String("target", "", "")
			cmd.Flags().String("editor", "", "")
			cmd.Flags().Bool("fix", false, "")
			cmd.Flags().String("fix-file", "", "")
			cmd.Flags().BoolP("numbers", "n", false, "")
			cmd.Flags().BoolP("clipboard", "b", false, "")
			cmd.Flags().BoolP("interactive", "i", false, "")
			
			// Set flag values
			for flag, value := range tt.flags {
				cmd.Flags().Set(flag, value)
			}
			for flag, value := range tt.boolFlags {
				if value {
					cmd.Flags().Set(flag, "true")
				}
			}
			
			result, err := buildRequestFromFlags(cmd, tt.args)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if result.BasePrompt != tt.expected.BasePrompt {
				t.Errorf("BasePrompt = %q, expected %q", result.BasePrompt, tt.expected.BasePrompt)
			}
			
			if result.PreTemplate != tt.expected.PreTemplate {
				t.Errorf("PreTemplate = %q, expected %q", result.PreTemplate, tt.expected.PreTemplate)
			}
			
			if result.PostTemplate != tt.expected.PostTemplate {
				t.Errorf("PostTemplate = %q, expected %q", result.PostTemplate, tt.expected.PostTemplate)
			}
			
			if result.Interactive != tt.expected.Interactive {
				t.Errorf("Interactive = %v, expected %v", result.Interactive, tt.expected.Interactive)
			}
			
			if result.FixMode != tt.expected.FixMode {
				t.Errorf("FixMode = %v, expected %v", result.FixMode, tt.expected.FixMode)
			}
			
			if result.NumberSelect != tt.expected.NumberSelect {
				t.Errorf("NumberSelect = %v, expected %v", result.NumberSelect, tt.expected.NumberSelect)
			}
			
			if result.FromClipboard != tt.expected.FromClipboard {
				t.Errorf("FromClipboard = %v, expected %v", result.FromClipboard, tt.expected.FromClipboard)
			}
			
			if result.ForceInteractive != tt.expected.ForceInteractive {
				t.Errorf("ForceInteractive = %v, expected %v", result.ForceInteractive, tt.expected.ForceInteractive)
			}
			
			if result.ForceNonInteractive != tt.expected.ForceNonInteractive {
				t.Errorf("ForceNonInteractive = %v, expected %v", result.ForceNonInteractive, tt.expected.ForceNonInteractive)
			}
		})
	}
}

// TestValidateRequest removed - validation is now handled by the orchestrator