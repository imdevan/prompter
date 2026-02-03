package workflow

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
)

const defaultFixTemplate = "Please evaluate the following output for errors, and resolve any issues. If there are multiple issues please fix them in order."

// Generator assembles prompts using templates and request data.
type Generator struct {
	Repo template.Repository
}

// NewGenerator returns a prompt generator.
func NewGenerator(repo template.Repository) *Generator {
	return &Generator{Repo: repo}
}

// Run assembles a prompt from templates, inputs, and config.
func (g *Generator) Run(req domain.Request, cfg domain.Config) (string, error) {
	if g.Repo == nil {
		return "", errors.New("template repository is required")
	}

	cwd := req.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	files, err := readFiles(req.Files)
	if err != nil {
		return "", err
	}

	var directory DirectoryInfo
	if req.IncludeDirectory {
		root := req.DirectoryPath
		if root == "" {
			root = cwd
		}
		strategy := req.DirectoryStrategy
		if strategy == "" {
			strategy = cfg.DirectoryStrategy
		}
		dirFiles, err := collectDirectoryFiles(root, strategy)
		if err != nil {
			return "", err
		}
		directory = DirectoryInfo{Root: root, Files: dirFiles}
	}

	data := TemplateData{
		BasePrompt: req.BasePrompt,
		Files:      files,
		Directory:  directory,
		Fix:        req.Fix,
		FixContent: req.Fix.Output,
		CWD:        cwd,
		Env:        req.Env,
		Config:     cfg,
	}
	// index template
	var parts []string
	appendPart := func(text string) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		parts = append(parts, text)
	}

	if indexTemplate, err := g.Repo.Get("index"); err == nil {
		data.Prompt = strings.Join(parts, "\n\n")
		rendered, err := template.Render(template.StripFrontmatter(indexTemplate.Content), data)
		if err != nil {
			return "", err
		}
		appendPart(rendered)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	agentTemplates, err := collectAgentTemplates(cwd, req.TemplateNames)
	if err != nil {
		return "", err
	}
	for _, content := range agentTemplates {
		data.Prompt = strings.Join(parts, "\n\n")
		rendered, err := template.Render(template.StripFrontmatter(content), data)
		if err != nil {
			return "", err
		}
		appendPart(rendered)
	}

	if req.Fix.Enabled {
		fixContent := defaultFixTemplate
		if fixTemplate, err := g.Repo.Get("fix"); err == nil {
			fixContent = fixTemplate.Content
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		data.Prompt = strings.Join(parts, "\n\n")
		rendered, err := template.Render(template.StripFrontmatter(fixContent), data)
		if err != nil {
			return "", err
		}
		appendPart(rendered)
	}

	order := buildTemplateOrder(req.TemplateOrder, req.TemplateNames)
	for _, name := range order {
		if name == domain.BasePromptToken {
			appendPart(req.BasePrompt)
			continue
		}
		if name == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "agents.md") {
			continue
		}
		tmpl, err := g.Repo.Get(name)
		if err != nil {
			return "", err
		}
		data.Prompt = strings.Join(parts, "\n\n")
		rendered, err := template.Render(template.StripFrontmatter(tmpl.Content), data)
		if err != nil {
			return "", err
		}
		appendPart(rendered)
	}

	if len(files) > 0 {
		appendPart(formatFiles("Files", files))
	}
	if req.IncludeDirectory && len(directory.Files) > 0 {
		appendPart(formatFiles("Directory", directory.Files))
	}
	if req.Fix.Enabled && strings.TrimSpace(req.Fix.Output) != "" {
		appendPart(fmt.Sprintf("Command Output:\n%s", strings.TrimSpace(req.Fix.Output)))
	}
	if strings.TrimSpace(req.PipedInput) != "" {
		if !req.Fix.Enabled || strings.TrimSpace(req.PipedInput) != strings.TrimSpace(req.Fix.Output) {
			appendPart(strings.TrimSpace(req.PipedInput))
		}
	}

	return strings.Join(parts, "\n\n"), nil
}

func buildTemplateOrder(order []string, templates []string) []string {
	if len(order) == 0 {
		order = append([]string{}, templates...)
	}
	if !containsToken(order, domain.BasePromptToken) {
		order = append(order, domain.BasePromptToken)
	}
	return order
}

func containsToken(items []string, token string) bool {
	for _, item := range items {
		if item == token {
			return true
		}
	}
	return false
}

// FileContent captures file data included in prompts.
type FileContent struct {
	Path    string
	Content string
}

// DirectoryInfo captures directory data included in prompts.
type DirectoryInfo struct {
	Root  string
	Files []FileContent
}

// TemplateData is the input for template rendering.
type TemplateData struct {
	Prompt     string
	BasePrompt string
	Files      []FileContent
	Directory  DirectoryInfo
	Fix        domain.FixInput
	FixContent string
	CWD        string
	Env        map[string]string
	Config     domain.Config
}

func readFiles(paths []string) ([]FileContent, error) {
	files := make([]FileContent, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		files = append(files, FileContent{
			Path:    path,
			Content: string(data),
		})
	}
	return files, nil
}

func collectDirectoryFiles(root, strategy string) ([]FileContent, error) {
	switch strings.ToLower(strategy) {
	case "git":
		return collectGitFiles(root)
	default:
		return collectFilesystemFiles(root)
	}
}

func collectGitFiles(root string) ([]FileContent, error) {
	cmd := exec.Command("git", "-C", root, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []FileContent
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		path := filepath.Join(root, line)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		files = append(files, FileContent{
			Path:    line,
			Content: string(data),
		})
	}
	return files, nil
}

func collectFilesystemFiles(root string) ([]FileContent, error) {
	if files, err := collectGitIgnoredFiles(root); err == nil {
		return files, nil
	}

	var files []FileContent
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, FileContent{
			Path:    rel,
			Content: string(data),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func collectGitIgnoredFiles(root string) ([]FileContent, error) {
	cmd := exec.Command("git", "-C", root, "ls-files", "-co", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []FileContent
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		path := filepath.Join(root, line)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		files = append(files, FileContent{
			Path:    line,
			Content: string(data),
		})
	}
	return files, nil
}

func collectAgentTemplates(cwd string, templateNames []string) ([]string, error) {
	if !containsTemplate(templateNames, "agents.md") {
		return nil, nil
	}
	var templates []string
	if content, err := readOptionalTemplate(filepath.Join(cwd, "AGENTS.md")); err != nil {
		return nil, err
	} else if content != "" {
		templates = append(templates, content)
	}
	return templates, nil
}

func containsTemplate(names []string, match string) bool {
	for _, name := range names {
		if strings.EqualFold(strings.TrimSpace(name), match) {
			return true
		}
	}
	return false
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

func formatFiles(label string, files []FileContent) string {
	var builder strings.Builder
	builder.WriteString(label)
	builder.WriteString(":\n")
	for _, file := range files {
		builder.WriteString("\n# ")
		builder.WriteString(file.Path)
		builder.WriteString("\n")
		builder.WriteString(file.Content)
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}
