package workflow

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
	"prompter-cli/internal/utils"
)

type skillSource struct {
	Token       string
	Prefix      string
	LocalDir    string
	LocalLabel  string
	GlobalDir   string
	GlobalLabel string
}

func skillSources() []skillSource {
	home, _ := os.UserHomeDir()
	configHome := utils.XDGConfigHome()
	claudeGlobal := ""
	agentsGlobal := ""
	opencodeGlobal := ""
	kiroGlobal := ""
	cursorGlobal := ""
	antigravityGlobal := ""
	if strings.TrimSpace(configHome) != "" {
		opencodeGlobal = filepath.Join(configHome, "opencode", "skills")
	}
	if strings.TrimSpace(home) != "" {
		claudeGlobal = filepath.Join(home, ".claude", "skills")
		agentsGlobal = filepath.Join(home, ".agents", "skills")
		kiroGlobal = filepath.Join(home, ".kiro", "skills")
		cursorGlobal = filepath.Join(home, ".cursor", "skills")
		antigravityGlobal = filepath.Join(home, ".antigravity", "skills")
	}
	return []skillSource{
		{
			Token:       "opencode",
			Prefix:      "opencode/skills",
			LocalDir:    filepath.Join(".opencode", "skills"),
			LocalLabel:  ".opencode/skills",
			GlobalDir:   opencodeGlobal,
			GlobalLabel: "opencode/skills",
		},
		{
			Token:       "claude",
			Prefix:      "claude/skills",
			LocalDir:    filepath.Join(".claude", "skills"),
			LocalLabel:  ".claude/skills",
			GlobalDir:   claudeGlobal,
			GlobalLabel: ".claude/skills",
		},
		{
			Token:       "agents",
			Prefix:      "agents/skills",
			LocalDir:    filepath.Join(".agents", "skills"),
			LocalLabel:  ".agents/skills",
			GlobalDir:   agentsGlobal,
			GlobalLabel: ".agents/skills",
		},
		{
			Token:       "kiro",
			Prefix:      "kiro/skills",
			LocalDir:    filepath.Join(".kiro", "skills"),
			LocalLabel:  ".kiro/skills",
			GlobalDir:   kiroGlobal,
			GlobalLabel: ".kiro/skills",
		},
		{
			Token:       "cursor",
			Prefix:      "cursor/skills",
			LocalDir:    filepath.Join(".cursor", "skills"),
			LocalLabel:  ".cursor/skills",
			GlobalDir:   cursorGlobal,
			GlobalLabel: ".cursor/skills",
		},
		{
			Token:       "antigravity",
			Prefix:      "antigravity/skills",
			LocalDir:    filepath.Join(".antigravity", "skills"),
			LocalLabel:  ".antigravity/skills",
			GlobalDir:   antigravityGlobal,
			GlobalLabel: ".antigravity/skills",
		},
	}
}

func opencodeTemplateRoots() []string {
	roots := make([]string, 0, 2)
	if configHome := strings.TrimSpace(utils.XDGConfigHome()); configHome != "" {
		roots = append(roots, filepath.Join(configHome, "opencode"))
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		roots = append(roots, filepath.Join(home, ".opencode"))
	}
	return roots
}

func collectSkillTemplatesForList(cwd, includeAgentsValue string, includeAll bool) ([]domain.Template, []domain.Template, error) {
	sources := skillSourcesForInclude(includeAgentsValue, includeAll)
	localRoots, err := projectSkillRoots(cwd)
	if err != nil {
		return nil, nil, err
	}
	local, err := collectSkillTemplatesFromRoots(localRoots, sources, true)
	if err != nil {
		return nil, nil, err
	}
	global, err := collectSkillTemplatesFromRoots(nil, sources, false)
	if err != nil {
		return nil, nil, err
	}
	return global, local, nil
}

func collectSkillTemplatesForSelection(cwd, includeAgentsValue string, includeAll bool) ([]domain.Template, error) {
	sources := skillSourcesForInclude(includeAgentsValue, includeAll)
	localRoots, err := projectSkillRoots(cwd)
	if err != nil {
		return nil, err
	}
	local, err := collectSkillTemplatesFromRoots(localRoots, sources, true)
	if err != nil {
		return nil, err
	}
	global, err := collectSkillTemplatesFromRoots(nil, sources, false)
	if err != nil {
		return nil, err
	}
	return append(local, global...), nil
}

func readSkillTemplate(cwd, name string) (string, error) {
	source, skillName := skillSourceForName(name)
	if source == nil {
		return "", nil
	}
	localRoots, err := projectSkillRoots(cwd)
	if err != nil {
		return "", err
	}
	content, err := readSkillTemplateFromRoots(localRoots, *source, skillName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(content) != "" {
		return content, nil
	}
	return readSkillTemplateFromRoots(nil, *source, skillName)
}

func skillSourcesForInclude(includeAgentsValue string, includeAll bool) []skillSource {
	sources := skillSources()
	if includeAll || strings.TrimSpace(strings.ToLower(includeAgentsValue)) == "all" {
		return sources
	}
	filtered := make([]skillSource, 0, len(sources))
	for _, source := range sources {
		if shouldIncludeAgent(includeAgentsValue, source.Token) {
			filtered = append(filtered, source)
		}
	}
	return filtered
}

func skillSourceForName(name string) (*skillSource, string) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, ""
	}
	path := strings.ToLower(filepath.ToSlash(trimmed))
	for _, source := range skillSources() {
		prefix := strings.ToLower(source.Prefix) + "/"
		if strings.HasPrefix(path, prefix) {
			skillName := strings.TrimPrefix(trimmed, source.Prefix+"/")
			skillName = strings.TrimSuffix(skillName, "/")
			return &source, skillName
		}
	}
	return nil, ""
}

func projectSkillRoots(cwd string) ([]string, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return nil, nil
	}
	stop, err := gitRoot(cwd)
	if err != nil {
		return nil, err
	}
	if stop == "" {
		return walkUpDirs(cwd), nil
	}
	return walkUpDirsUntil(cwd, stop), nil
}

func gitRoot(cwd string) (string, error) {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

func walkUpDirsUntil(start, stop string) []string {
	start = filepath.Clean(start)
	stop = filepath.Clean(stop)
	dirs := []string{}
	dir := start
	for {
		dirs = append(dirs, dir)
		if dir == stop {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dirs
}

func walkUpDirs(start string) []string {
	start = filepath.Clean(start)
	dirs := []string{}
	dir := start
	for {
		dirs = append(dirs, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dirs
}

func collectSkillTemplatesFromRoots(roots []string, sources []skillSource, local bool) ([]domain.Template, error) {
	var templates []domain.Template
	seen := make(map[string]bool)
	for _, source := range sources {
		if local {
			for _, root := range roots {
				dir := filepath.Join(root, source.LocalDir)
				entries, err := collectSkillsFromDir(dir, source.Prefix, source.LocalLabel)
				if err != nil {
					return nil, err
				}
				appendSkills(entries, seen, &templates)
			}
			continue
		}
		if source.Token == "opencode" {
			for _, dir := range opencodeGlobalSkillDirs() {
				entries, err := collectSkillsFromDir(dir.dir, source.Prefix, dir.label)
				if err != nil {
					return nil, err
				}
				appendSkills(entries, seen, &templates)
			}
			continue
		}
		if strings.TrimSpace(source.GlobalDir) == "" {
			continue
		}
		entries, err := collectSkillsFromDir(source.GlobalDir, source.Prefix, source.GlobalLabel)
		if err != nil {
			return nil, err
		}
		appendSkills(entries, seen, &templates)
	}
	return templates, nil
}

type skillDir struct {
	dir   string
	label string
}

func opencodeGlobalSkillDirs() []skillDir {
	dirs := []skillDir{}
	if configHome := strings.TrimSpace(utils.XDGConfigHome()); configHome != "" {
		dirs = append(dirs, skillDir{
			dir:   filepath.Join(configHome, "opencode", "skills"),
			label: "opencode/skills",
		})
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		dirs = append(dirs, skillDir{
			dir:   filepath.Join(home, ".opencode", "skills"),
			label: ".opencode/skills",
		})
	}
	return dirs
}

func appendSkills(entries []domain.Template, seen map[string]bool, templates *[]domain.Template) {
	for _, entry := range entries {
		key := strings.TrimSpace(strings.ToLower(entry.Name))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		*templates = append(*templates, entry)
	}
}

func collectSkillsFromDir(dir, prefix, label string) ([]domain.Template, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, nil
	}
	info, err := os.Stat(dir)
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
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != "SKILL.md" {
			return nil
		}
		relDir, err := filepath.Rel(dir, filepath.Dir(path))
		if err != nil {
			return err
		}
		if relDir == "." {
			relDir = ""
		}
		name := filepath.ToSlash(filepath.Join(prefix, relDir))
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		displayName := skillFrontmatterName(string(data))
		if displayName == "" {
			displayName = filepath.Base(relDir)
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

func readSkillTemplateFromRoots(roots []string, source skillSource, skillName string) (string, error) {
	rel := filepath.FromSlash(skillName)
	rel = filepath.Join(rel, "SKILL.md")
	var scanRoots []string
	baseDir := source.GlobalDir
	if roots != nil {
		scanRoots = roots
	} else {
		scanRoots = []string{""}
	}
	for _, root := range scanRoots {
		dir := baseDir
		if roots != nil {
			dir = filepath.Join(root, source.LocalDir)
		} else if source.Token == "opencode" {
			for _, globalDir := range opencodeGlobalSkillDirs() {
				path := filepath.Join(globalDir.dir, rel)
				if content, err := readOptionalTemplate(path); err != nil {
					return "", err
				} else if strings.TrimSpace(content) != "" {
					return content, nil
				}
			}
			continue
		} else if strings.TrimSpace(dir) == "" {
			continue
		}
		path := filepath.Join(dir, rel)
		if content, err := readOptionalTemplate(path); err != nil {
			return "", err
		} else if strings.TrimSpace(content) != "" {
			return content, nil
		}
	}
	return "", nil
}

func readOpencodeCommandTemplate(rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	for _, root := range opencodeTemplateRoots() {
		if strings.TrimSpace(root) == "" {
			continue
		}
		path := filepath.Join(root, "commands", rel)
		if content, err := readOptionalTemplate(path); err != nil {
			return "", err
		} else if strings.TrimSpace(content) != "" {
			return content, nil
		}
	}
	return "", nil
}

func skillFrontmatterName(content string) string {
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

func frontmatterHeader(content string) (string, bool) {
	trimmed := strings.TrimLeft(content, "\ufeff\r\n\t ")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return "", false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", false
	}
	return strings.Join(lines[1:end], "\n"), true
}

func readOptionalTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return template.StripFrontmatter(string(data)), nil
}
