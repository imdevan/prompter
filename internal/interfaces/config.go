package interfaces

// Config represents the application configuration
type Config struct {
	PromptsLocation   string `toml:"prompts_location"`
	Editor            string `toml:"editor"`
	DefaultPre        string `toml:"default_pre"`
	DefaultPost       string `toml:"default_post"`
	FixFile           string `toml:"fix_file"`
	MaxFileSizeBytes  int64  `toml:"max_file_size_bytes"`
	MaxTotalBytes     int64  `toml:"max_total_bytes"`
	AllowOversize     bool   `toml:"allow_oversize"`
	DirectoryStrategy string `toml:"directory_strategy"`
	Target            string `toml:"target"`
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