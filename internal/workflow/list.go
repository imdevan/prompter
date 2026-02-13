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
	Heading   string
	Location  string
	Templates []domain.Template
}

// ListOptions configures list behavior.
type ListOptions struct {
	IncludeAgents bool
}

// ListTemplates collects templates grouped by local and global locations.
func ListTemplates(cwd string, cfg domain.Config, opts ListOptions) ([]TemplateGroup, error) {
	var groups []TemplateGroup

	if cfg.LocalPromptsLocation != "" {
		localPath := filepath.Join(cwd, cfg.LocalPromptsLocation)
		templates, err := template.NewRepository(localPath).List()
		if err != nil {
			return nil, err
		}
		if len(templates) > 0 {
			groups = append(groups, TemplateGroup{
				Label:     "Local",
				Location:  localPath,
				Templates: templates,
			})
		}
	}

	if cfg.PromptsLocation != "" {
		templates, err := template.NewRepository(cfg.PromptsLocation).List()
		if err != nil {
			return nil, err
		}
		if len(templates) > 0 {
			groups = append(groups, TemplateGroup{
				Label:     "Global",
				Location:  cfg.PromptsLocation,
				Templates: templates,
			})
		}
	}

	if includeAgents(cfg.IncludeAgents) || opts.IncludeAgents {
		globalSkills, localSkills, err := collectSkillTemplatesForList(cwd, cfg.IncludeAgents, opts.IncludeAgents)
		if err != nil {
			return nil, err
		}
		cursorCommands, err := collectCursorCommandTemplates(cwd, cfg.IncludeAgents, opts.IncludeAgents)
		if err != nil {
			return nil, err
		}
		if len(cursorCommands) > 0 {
			localSkills = append(localSkills, cursorCommands...)
		}
		if len(globalSkills) > 0 {
			groups = append(groups, TemplateGroup{
				Heading:   "Global Skills",
				Templates: globalSkills,
			})
		}
		if len(localSkills) > 0 {
			groups = append(groups, TemplateGroup{
				Heading:   "Local Skills",
				Templates: localSkills,
			})
		}

		agents, err := collectAgentTemplatesForList(cwd, cfg.IncludeAgents, opts.IncludeAgents)
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

func collectAgentTemplatesForList(cwd, includeAgentsValue string, includeAll bool) ([]domain.Template, error) {
	var templates []domain.Template
	if includeAll || shouldIncludeAgent(includeAgentsValue, "agents") || shouldIncludeAgent(includeAgentsValue, "agents.md") {
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
	if includeAll || shouldIncludeAgent(includeAgentsValue, "kiro") {
		kiroDir := filepath.Join(cwd, ".kiro", "steering")
		kiroTemplates, err := collectTemplatesFromDir(kiroDir, "kiro/steering", "kiro/steering")
		if err != nil {
			return nil, err
		}
		templates = append(templates, kiroTemplates...)
	}
	if includeAll || shouldIncludeAgent(includeAgentsValue, "opencode") {
		opencodeRoots := opencodeTemplateRoots()
		opencodeCommands, err := collectTemplatesFromDirs(opencodeRoots, "commands", "opencode/commands")
		if err != nil {
			return nil, err
		}
		templates = append(templates, opencodeCommands...)
	}
	return templates, nil
}

func collectCursorCommandTemplates(cwd, includeAgentsValue string, includeAll bool) ([]domain.Template, error) {
	if !includeAll && !shouldIncludeAgent(includeAgentsValue, "cursor") {
		return nil, nil
	}
	cursorDir := filepath.Join(cwd, ".cursor", "commands")
	return collectTemplatesFromDir(cursorDir, "cursor/commands", ".cursor/commands")
}

func templateFrontmatterName(content string) string {
	header, ok := frontmatterHeader(content)
	if !ok {
		return ""
	}
	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.TrimSpace(strings.ToLower(key)) != "name" {
			continue
		}
		value = strings.TrimSpace(strings.Trim(value, "\""))
		return value
	}
	return ""
}

func agentTemplateFromFile(cwd, filename string) (*domain.Template, error) {
	path := filepath.Join(cwd, filename)
	if _, err := os.Stat(path); err == nil {
		return &domain.Template{
			Name:        "agents.md",
			DisplayName: "Agent instructions",
			Description: "From " + filename,
			Location:    path,
		}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return nil, nil
}

func includeAgents(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return false
	}
	if value == "none" {
		return false
	}
	return true
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

func collectTemplatesFromDir(root, namePrefix, label string) ([]domain.Template, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}
	var templates []domain.Template
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(namePrefix, rel))
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		displayName := templateFrontmatterName(string(data))
		if displayName == "" {
			displayName = strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		}
		templates = append(templates, domain.Template{
			Name:        name,
			DisplayName: displayName,
			Description: "From " + label,
			Location:    path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func collectTemplatesFromDirs(roots []string, subdir, labelPrefix string) ([]domain.Template, error) {
	var templates []domain.Template
	seen := make(map[string]bool)
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		dir := filepath.Join(root, subdir)
		items, err := collectTemplatesFromDir(dir, labelPrefix, labelPrefix)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			key := strings.TrimSpace(strings.ToLower(item.Name))
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			templates = append(templates, item)
		}
	}
	return templates, nil
}

func AgentTemplatesForSelection(cwd string, cfg domain.Config, includeAll bool) ([]domain.Template, error) {
	if !includeAgents(cfg.IncludeAgents) && !includeAll {
		return nil, nil
	}
	templates, err := collectAgentTemplatesForList(cwd, cfg.IncludeAgents, includeAll)
	if err != nil {
		return nil, err
	}
	skills, err := collectSkillTemplatesForSelection(cwd, cfg.IncludeAgents, includeAll)
	if err != nil {
		return nil, err
	}
	cursorCommands, err := collectCursorCommandTemplates(cwd, cfg.IncludeAgents, includeAll)
	if err != nil {
		return nil, err
	}
	if len(cursorCommands) > 0 {
		skills = append(skills, cursorCommands...)
	}
	if len(skills) > 0 {
		templates = append(templates, skills...)
	}
	return templates, nil
}
