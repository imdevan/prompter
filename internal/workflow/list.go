package workflow

import (
	"os"
	"path/filepath"
	"strings"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
)

// TemplateGroup groups templates by location for listing.
type TemplateGroup struct {
	Label     string
	Location  string
	Templates []domain.Template
}

// ListTemplates collects templates grouped by local and global locations.
func ListTemplates(cwd string, cfg domain.Config) ([]TemplateGroup, error) {
	var groups []TemplateGroup

	if cfg.LocalPromptsLocation != "" {
		localPath := filepath.Join(cwd, cfg.LocalPromptsLocation)
		templates, err := template.NewRepository(localPath).List()
		if err != nil {
			return nil, err
		}
		groups = append(groups, TemplateGroup{
			Label:     "Local",
			Location:  localPath,
			Templates: templates,
		})
	}

	if cfg.PromptsLocation != "" {
		templates, err := template.NewRepository(cfg.PromptsLocation).List()
		if err != nil {
			return nil, err
		}
		groups = append(groups, TemplateGroup{
			Label:     "Global",
			Location:  cfg.PromptsLocation,
			Templates: templates,
		})
	}

	if includeAgents(cfg.IncludeAgents) {
		agents, err := collectAgentTemplatesForList(cwd, cfg.IncludeAgents)
		if err != nil {
			return nil, err
		}
		if len(agents) > 0 {
			groups = append(groups, TemplateGroup{
				Label:     "Agent",
				Location:  "",
				Templates: agents,
			})
		}
	}

	return groups, nil
}

func collectAgentTemplatesForList(cwd, includeAgentsValue string) ([]domain.Template, error) {
	var templates []domain.Template
	if shouldIncludeAgent(includeAgentsValue, "agents") || shouldIncludeAgent(includeAgentsValue, "agents.md") {
		if tmpl, err := agentTemplateFromFile(cwd, "AGENTS.md"); err != nil {
			return nil, err
		} else if tmpl != nil {
			templates = append(templates, *tmpl)
		} else if tmpl, err := agentTemplateFromFile(cwd, "agents.md"); err != nil {
			return nil, err
		} else if tmpl != nil {
			templates = append(templates, *tmpl)
		}
	}
	return templates, nil
}

func agentTemplateFromFile(cwd, filename string) (*domain.Template, error) {
	path := filepath.Join(cwd, filename)
	if _, err := os.Stat(path); err == nil {
		return &domain.Template{
			Name:        "agents.md",
			Title:       "Agent instructions",
			Description: "From " + filename,
			Location:    path,
		}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return nil, nil
}

func includeAgents(value string) bool {
	return !shouldIncludeAgent(value, "none")
}

func shouldIncludeAgent(value, token string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return false
	}
	if value == "all" {
		return true
	}
	token = strings.ToLower(token)
	for _, part := range splitConfigList(value) {
		if part == token {
			return true
		}
	}
	return false
}

func splitConfigList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})
}
