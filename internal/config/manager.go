package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"prompter-cli/internal/interfaces"
)

// Manager implements the ConfigManager interface
type Manager struct {
	v     *viper.Viper
	flags map[string]interface{} // Store flag values for precedence
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetEnvPrefix("PROMPTER")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// Set defaults
	setDefaults(v)
	
	return &Manager{
		v:     v,
		flags: make(map[string]interface{}),
	}
}

// SetConfigPath sets the configuration file path
func (m *Manager) SetConfigPath(path string) {
	if path != "" {
		m.v.SetConfigFile(expandPath(path))
	}
}

// setDefaults sets the default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("prompts_location", "~/.config/prompter")
	v.SetDefault("editor", "nvim")
	v.SetDefault("default_pre", "")
	v.SetDefault("default_post", "")
	v.SetDefault("fix_file", "/tmp/prompter-fix.txt")
	v.SetDefault("directory_strategy", "git")
	v.SetDefault("target", "clipboard")
}

// Load loads configuration from the specified path
func (m *Manager) Load(path string) (*interfaces.Config, error) {
	if path == "" {
		// Use default config path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(homeDir, ".config", "prompter", "config.toml")
	}
	

	
	// Expand tilde in path
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	}
	
	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults
		return m.getConfigFromViper(), nil
	}
	
	m.v.SetConfigFile(path)
	
	if err := m.v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}
	
	return m.getConfigFromViper(), nil
}

// SetFlag sets a flag value for precedence resolution
func (m *Manager) SetFlag(key string, value interface{}) {
	m.flags[key] = value
}

// Resolve applies precedence rules (flags > env > config > defaults)
func (m *Manager) Resolve() (*interfaces.Config, error) {
	config := m.getConfigFromViper()
	
	// Apply flag overrides (highest precedence)
	m.applyFlagOverrides(config)
	
	return config, nil
}

// applyFlagOverrides applies flag values over the configuration
func (m *Manager) applyFlagOverrides(config *interfaces.Config) {
	if val, exists := m.flags["prompts_location"]; exists && val != nil {
		if str, ok := val.(string); ok && str != "" {
			config.PromptsLocation = expandPath(str)
		}
	}
	
	if val, exists := m.flags["editor"]; exists && val != nil {
		if str, ok := val.(string); ok && str != "" {
			config.Editor = str
		}
	}
	
	if val, exists := m.flags["default_pre"]; exists && val != nil {
		if str, ok := val.(string); ok && str != "" {
			config.DefaultPre = str
		}
	}
	
	if val, exists := m.flags["default_post"]; exists && val != nil {
		if str, ok := val.(string); ok && str != "" {
			config.DefaultPost = str
		}
	}
	
	if val, exists := m.flags["fix_file"]; exists && val != nil {
		if str, ok := val.(string); ok && str != "" {
			config.FixFile = expandPath(str)
		}
	}
	

	
	if val, exists := m.flags["directory_strategy"]; exists && val != nil {
		if str, ok := val.(string); ok && str != "" {
			config.DirectoryStrategy = str
		}
	}
	
	if val, exists := m.flags["target"]; exists && val != nil {
		if str, ok := val.(string); ok && str != "" {
			config.Target = str
		}
	}
}

// Validate validates the configuration values
func (m *Manager) Validate(config *interfaces.Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	

	
	// Validate directory strategy
	validStrategies := map[string]bool{
		"git":        true,
		"filesystem": true,
	}
	if !validStrategies[config.DirectoryStrategy] {
		return fmt.Errorf("invalid directory_strategy: %s (must be 'git' or 'filesystem')", config.DirectoryStrategy)
	}
	
	// Validate target
	validTargets := map[string]bool{
		"clipboard": true,
		"stdout":    true,
	}
	// Also allow file: prefix
	if !validTargets[config.Target] && !strings.HasPrefix(config.Target, "file:") {
		return fmt.Errorf("invalid target: %s (must be 'clipboard', 'stdout', or 'file:/path')", config.Target)
	}
	
	// Validate prompts location exists or can be created
	if config.PromptsLocation != "" {
		expandedPath := expandPath(config.PromptsLocation)
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(expandedPath, 0755); err != nil {
				return fmt.Errorf("prompts_location directory does not exist and cannot be created: %s", expandedPath)
			}
		}
	}
	
	return nil
}

// getConfigFromViper converts viper configuration to Config struct
// This handles env > config > defaults precedence (flags are applied separately)
func (m *Manager) getConfigFromViper() *interfaces.Config {
	return &interfaces.Config{
		PromptsLocation:   expandPath(m.v.GetString("prompts_location")),
		Editor:            m.v.GetString("editor"),
		DefaultPre:        m.v.GetString("default_pre"),
		DefaultPost:       m.v.GetString("default_post"),
		FixFile:           expandPath(m.v.GetString("fix_file")),
		DirectoryStrategy: m.v.GetString("directory_strategy"),
		Target:            m.v.GetString("target"),
	}
}

// MergeConfig merges another configuration into this manager
func (m *Manager) MergeConfig(other *interfaces.Config) {
	if other == nil {
		return
	}
	
	if other.PromptsLocation != "" {
		m.v.Set("prompts_location", other.PromptsLocation)
	}
	if other.Editor != "" {
		m.v.Set("editor", other.Editor)
	}
	if other.DefaultPre != "" {
		m.v.Set("default_pre", other.DefaultPre)
	}
	if other.DefaultPost != "" {
		m.v.Set("default_post", other.DefaultPost)
	}
	if other.FixFile != "" {
		m.v.Set("fix_file", other.FixFile)
	}

	if other.DirectoryStrategy != "" {
		m.v.Set("directory_strategy", other.DirectoryStrategy)
	}
	if other.Target != "" {
		m.v.Set("target", other.Target)
	}
}

// expandPath expands ~ to user home directory
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path // Return original path if we can't get home dir
	}
	
	return filepath.Join(homeDir, path[2:])
}