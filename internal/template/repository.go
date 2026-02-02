package template

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"prompter-cli/internal/domain"
)

// RepositoryFS loads templates from one or more directories.
type RepositoryFS struct {
	Locations []string
}

// NewRepository builds a filesystem-backed repository.
func NewRepository(locations ...string) *RepositoryFS {
	return &RepositoryFS{Locations: locations}
}

// List returns templates from configured locations, preferring earlier locations on name conflicts.
func (r *RepositoryFS) List() ([]domain.Template, error) {
	seen := make(map[string]struct{})
	var templates []domain.Template

	for _, location := range r.Locations {
		if location == "" {
			continue
		}
		entries, err := os.ReadDir(location)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if filepath.Ext(name) != ".md" {
				continue
			}
			stem := strings.TrimSuffix(name, ".md")
			if _, ok := seen[stem]; ok {
				continue
			}
			path := filepath.Join(location, name)
			template, err := loadTemplateFile(path)
			if err != nil {
				return nil, err
			}
			template.Location = location
			templates = append(templates, template)
			seen[stem] = struct{}{}
		}
	}

	sort.Slice(templates, func(i, j int) bool {
		if templates[i].Pinned != templates[j].Pinned {
			return templates[i].Pinned
		}
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}

// Get returns a template by name, searching locations in order.
func (r *RepositoryFS) Get(name string) (domain.Template, error) {
	for _, location := range r.Locations {
		if location == "" {
			continue
		}
		path := filepath.Join(location, name+".md")
		template, err := loadTemplateFile(path)
		if err == nil {
			template.Location = location
			return template, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return domain.Template{}, err
	}
	return domain.Template{}, os.ErrNotExist
}

// Save writes a template to the first configured location.
func (r *RepositoryFS) Save(template domain.Template) error {
	if len(r.Locations) == 0 || r.Locations[0] == "" {
		return errors.New("no template locations configured")
	}
	path := filepath.Join(r.Locations[0], template.Name+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(template.Content), 0o644)
}

func loadTemplateFile(path string) (domain.Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Template{}, err
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	template := domain.Template{
		Name:    name,
		Content: string(data),
	}
	parsed, err := parseFrontmatter(template)
	if err != nil {
		return domain.Template{}, err
	}
	return parsed, nil
}

func parseFrontmatter(template domain.Template) (domain.Template, error) {
	header, body, ok := splitFrontmatter(template.Content)
	if !ok {
		return template, nil
	}
	template.Content = body

	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")
		switch key {
		case "title":
			template.Title = value
		case "description":
			template.Description = value
		case "flag":
			template.Flag = value
		case "shorthand":
			template.Shorthand = value
		case "pin":
			template.Pinned = value == "true"
		}
	}

	return template, nil
}

// StripFrontmatter removes YAML frontmatter from a template's content.
func StripFrontmatter(content string) string {
	_, body, ok := splitFrontmatter(content)
	if !ok {
		return content
	}
	return body
}

func splitFrontmatter(content string) (string, string, bool) {
	trimmed := strings.TrimLeft(content, "\ufeff\r\n\t ")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return "", content, false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", content, false
	}
	header := strings.Join(lines[1:end], "\n")
	body := strings.Join(lines[end+1:], "\n")
	body = strings.TrimLeft(body, "\r\n")
	return header, body, true
}
