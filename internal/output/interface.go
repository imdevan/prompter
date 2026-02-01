package output

// Clipboard writes content to the system clipboard.
type Clipboard interface {
	WriteText(text string) error
}

// Editor opens a file in the user's editor.
type Editor interface {
	Open(path string) error
}
