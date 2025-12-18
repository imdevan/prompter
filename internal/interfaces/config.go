package interfaces

// Config represents the application configuration
type Config struct {
	PromptsLocation      string `toml:"prompts_location"`
	LocalPromptsLocation string `toml:"local_prompts_location"`
	Editor               string `toml:"editor"`
	DefaultPre           string `toml:"default_pre"`
	DefaultPost          string `toml:"default_post"`
	FixFile              string `toml:"fix_file"`
	DirectoryStrategy    string `toml:"directory_strategy"`
	Target               string `toml:"target"`
	InteractiveDefault   bool   `toml:"interactive_default"`
}

// ConfigManager handles configuration loading and resolution
type ConfigManager interface {
	// Load loads configuration from the specified path
	Load(path string) (*Config, error)
	
	// Resolve applies precedence rules (flags > env > config > defaults)
	Resolve() (*Config, error)
	
	// Validate validates the configuration values
	Validate(config *Config) error
}