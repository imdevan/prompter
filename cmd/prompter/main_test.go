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
				BasePrompt:  "test prompt",
				Interactive: false,
				Files:       []string{},
			},
		},
		{
			name: "fix mode",
			boolFlags: map[string]bool{
				"fix": true,
				"yes": true,
			},
			expected: &models.PromptRequest{
				FixMode:     true,
				Interactive: false,
				Files:       []string{},
			},
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
			cmd.Flags().String("directory", "", "")
			cmd.Flags().String("target", "", "")
			cmd.Flags().String("editor", "", "")
			cmd.Flags().Bool("fix", false, "")
			
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
		})
	}
}

// TestValidateRequest removed - validation is now handled by the orchestrator