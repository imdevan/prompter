package models

// PromptRequest represents the main application request with all user inputs
type PromptRequest struct {
	BasePrompt        string   `json:"base_prompt"`
	PreTemplate       string   `json:"pre_template"`
	PostTemplate      string   `json:"post_template"`
	Files             []string `json:"files"`
	Directory         string   `json:"directory"`
	FixMode           bool     `json:"fix_mode"`
	FixFile           string   `json:"fix_file"`
	Target            string   `json:"target"`
	Editor            string   `json:"editor"`
	EditorRequested   bool     `json:"editor_requested"`   // Track if --editor flag was explicitly used
	Interactive       bool     `json:"interactive"`
	ConfigPath        string   `json:"config_path"`
	NumberSelect      bool     `json:"number_select"`      // Enable number key selection for templates
	FromClipboard     bool     `json:"from_clipboard"`     // Read base prompt from clipboard
	ForceInteractive  bool     `json:"force_interactive"`  // -i flag was used
	ForceNonInteractive bool   `json:"force_non_interactive"` // -y flag was used
}

// NewPromptRequest creates a new PromptRequest with default values
func NewPromptRequest() *PromptRequest {
	return &PromptRequest{
		Interactive: true, // Default to interactive mode
		Files:       []string{},
	}
}