package domain

import "strings"

// Template represents a prompt template and its metadata.
type Template struct {
	Name        string
	DisplayName string
	Content     string
	Description string
	Flag        string
	Shorthand   string
	Pinned      bool
	Location    string
}

// DisplayLabel returns the user-facing template name, preferring a frontmatter override.
func (t Template) DisplayLabel() string {
	if strings.TrimSpace(t.DisplayName) != "" {
		return strings.TrimSpace(t.DisplayName)
	}
	if strings.TrimSpace(t.Name) != "" {
		return strings.TrimSpace(t.Name)
	}
	return ""
}
