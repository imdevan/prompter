package models

// PromptRequest represents the main application state for a prompt generation request
type PromptRequest struct {
	BasePrompt    string
	PreTemplate   string
	PostTemplate  string
	Files         []string
	Directory     string
	FixMode       bool
	FixFile       string
	Target        string
	Editor        string
	Interactive   bool
}