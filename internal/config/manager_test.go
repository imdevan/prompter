package config

import (
	"os"
	"path/filepath"
	"testing"

	"prompter-cli/internal/interfaces"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
	if manager.v == nil {
		t.Fatal("NewManager() created manager with nil viper instance")
	}
}

func TestManager_Load_DefaultPath(t *testing.T) {
	manager := NewManager()
	
	// Test loading with empty path (should use defaults)
	config, err := manager.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") failed: %v", err)
	}
	
	// Verify defaults are set
	if config.DirectoryStrategy != "git" {
		t.Errorf("Expected DirectoryStrategy to be 'git', got %s", config.DirectoryStrategy)
	}
	if config.Target != "clipboard" {
		t.Errorf("Expected Target to be 'clipboard', got %s", config.Target)
	}
}

func TestManager_Load_CustomFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	
	configContent := `
prompts_location = "/custom/prompts"
editor = "vim"
default_pre = "custom_pre"
default_post = "custom_post"
fix_file = "/custom/fix.txt"

directory_strategy = "filesystem"
target = "stdout"
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	manager := NewManager()
	config, err := manager.Load(configPath)
	if err != nil {
		t.Fatalf("Load(%s) failed: %v", configPath, err)
	}
	
	// Verify custom values are loaded
	if config.PromptsLocation != "/custom/prompts" {
		t.Errorf("Expected PromptsLocation to be '/custom/prompts', got %s", config.PromptsLocation)
	}
	if config.Editor != "vim" {
		t.Errorf("Expected Editor to be 'vim', got %s", config.Editor)
	}

}

func TestManager_Validate(t *testing.T) {
	manager := NewManager()
	
	tests := []struct {
		name    string
		config  *interfaces.Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &interfaces.Config{
				PromptsLocation:   "/tmp/prompts",
				DirectoryStrategy: "git",
				Target:            "clipboard",
			},
			wantErr: false,
		},
		{
			name: "invalid directory strategy",
			config: &interfaces.Config{
				DirectoryStrategy: "invalid",
				Target:            "clipboard",
			},
			wantErr: true,
		},
		{
			name: "invalid target",
			config: &interfaces.Config{
				DirectoryStrategy: "git",
				Target:            "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid file target",
			config: &interfaces.Config{
				DirectoryStrategy: "git",
				Target:            "file:/tmp/output.txt",
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_SetFlag(t *testing.T) {
	manager := NewManager()
	
	manager.SetFlag("editor", "vim")
	manager.SetFlag("target", "stdout")
	
	if manager.flags["editor"] != "vim" {
		t.Errorf("Expected flag 'editor' to be 'vim', got %v", manager.flags["editor"])
	}
	if manager.flags["target"] != "stdout" {
		t.Errorf("Expected flag 'target' to be 'stdout', got %v", manager.flags["target"])
	}
}

func TestManager_Resolve_FlagPrecedence(t *testing.T) {
	// Create a temporary config file with some values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	
	configContent := `
editor = "nano"
target = "stdout"
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	manager := NewManager()
	
	// Load config file
	_, err = manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Set flags that should override config values
	manager.SetFlag("editor", "vim")
	// Don't set target flag so it remains from config
	
	// Resolve should apply flag precedence
	config, err := manager.Resolve()
	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}
	
	// Verify flags override config values
	if config.Editor != "vim" {
		t.Errorf("Expected Editor to be 'vim' (from flag), got %s", config.Editor)
	}

	// Target should remain from config since no flag was set
	if config.Target != "stdout" {
		t.Errorf("Expected Target to be 'stdout' (from config), got %s", config.Target)
	}
}

func TestManager_Resolve_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("PROMPTER_EDITOR", "emacs")
	os.Setenv("PROMPTER_TARGET", "stdout")
	defer func() {
		os.Unsetenv("PROMPTER_EDITOR")
		os.Unsetenv("PROMPTER_TARGET")
	}()
	
	manager := NewManager()
	
	config, err := manager.Resolve()
	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}
	
	// Verify environment variables are used
	if config.Editor != "emacs" {
		t.Errorf("Expected Editor to be 'emacs' (from env), got %s", config.Editor)
	}

}

func TestManager_MergeConfig(t *testing.T) {
	manager := NewManager()
	
	other := &interfaces.Config{
		Editor: "vim",
		Target: "stdout",
	}
	
	manager.MergeConfig(other)
	
	config := manager.getConfigFromViper()
	
	if config.Editor != "vim" {
		t.Errorf("Expected Editor to be 'vim', got %s", config.Editor)
	}

	if config.Target != "stdout" {
		t.Errorf("Expected Target to be 'stdout', got %s", config.Target)
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "absolute path",
			path:     "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			path:     "relative/path",
			expected: "relative/path",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.path)
			if result != tt.expected {
				t.Errorf("expandPath(%s) = %s, expected %s", tt.path, result, tt.expected)
			}
		})
	}
	
	// Test tilde expansion separately since it depends on user home
	homeDir, err := os.UserHomeDir()
	if err == nil {
		result := expandPath("~/test/path")
		expected := filepath.Join(homeDir, "test/path")
		if result != expected {
			t.Errorf("expandPath(~/test/path) = %s, expected %s", result, expected)
		}
	}
}