package domain

// Template represents a prompt template and its metadata.
type Template struct {
	Name        string
	Content     string
	Title       string
	Description string
	Flag        string
	Shorthand   string
	Pinned      bool
	Location    string
}
