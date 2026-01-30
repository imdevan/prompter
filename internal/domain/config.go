package domain

// Config describes the resolved configuration.
type Config struct {
	PromptsLocation      string
	HistoryLocation      string
	LocalPromptsLocation string
	IncludeAgents        string
	Editor               string
	DirectoryStrategy    string
	Target               string
	InteractiveDefault   bool
}

// DefaultConfig returns the default configuration values.
func DefaultConfig() Config {
	return Config{}
}
