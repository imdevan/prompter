package interfaces

// OutputHandler manages different output destinations
type OutputHandler interface {
	// WriteToClipboard copies content to the system clipboard
	WriteToClipboard(content string) error
	
	// WriteToStdout writes content to standard output
	WriteToStdout(content string) error
	
	// WriteToFile writes content to the specified file path
	WriteToFile(content string, path string) error
	
	// OpenInEditor opens content in the specified editor
	OpenInEditor(content string, editor string) error
}