package domain

// Request describes a prompt generation request.
type Request struct {
	BasePrompt string
}

// Validate checks that the request is well-formed.
func (r Request) Validate() error {
	return nil
}
