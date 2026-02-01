package domain

// FixInput captures fix-mode data.
type FixInput struct {
	Enabled bool
	Command string
	Output  string
}

// Request describes a prompt generation request.
type Request struct {
	BasePrompt        string
	TemplateNames     []string
	Files             []string
	IncludeDirectory  bool
	DirectoryPath     string
	DirectoryStrategy string
	Fix               FixInput
	Target            string
	PipedInput        string
	CWD               string
	Env               map[string]string
}

// Validate checks that the request is well-formed.
func (r Request) Validate() error {
	return nil
}
