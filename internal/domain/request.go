package domain

// Request describes a prompt generation request.
type Request struct {
	BasePrompt        string
	TemplateNames     []string
	TemplateOrder     []string
	HistorySuffix     string
	HistoryTag        string
	Files             []string
	IncludeDirectory  bool
	DirectoryPath     string
	DirectoryStrategy string
	Target            string
	EditorTarget      bool
	EditorIsVim       bool
	PipedInput        string
	CWD               string
	Env               map[string]string
}

// BasePromptToken marks where the base prompt should be inserted in TemplateOrder.
const BasePromptToken = "__base_prompt__"

// Validate checks that the request is well-formed.
func (r Request) Validate() error {
	return nil
}
