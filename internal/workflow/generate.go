package workflow

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
)

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
		CWD:        cwd,
		Env:        req.Env,
		Config:     cfg,
	}
	assembled := strings.TrimSpace(req.BasePrompt)
	applyTemplate := func(content string) error {
		content = template.StripFrontmatter(content)
		hasPrompt := strings.Contains(content, ".Prompt")
		if hasPrompt {
			token := "__PROMPTER_PROMPT_TOKEN__"
			data.Prompt = token
			rendered, err := template.Render(content, data)
			if err != nil {
				return err
			}
			if strings.Contains(rendered, token) {
				parts := strings.Split(rendered, token)
				before := strings.Join(parts[:1], "")
				after := strings.Join(parts[1:], token)
				assembled = joinParts(before, assembled)
				assembled = joinParts(assembled, after)
				return nil
			}
			assembled = strings.TrimSpace(rendered)
			return nil
		}
		data.Prompt = assembled
		rendered, err := template.Render(content, data)
		if err != nil {
			return err
		}
		assembled = joinParts(rendered, assembled)
		return nil
	}

	templates, err := collectTemplateContents(cwd, g.Repo, req)
	if err != nil {
		return "", err
	}
	for i := len(templates) - 1; i >= 0; i-- {
		if err := applyTemplate(templates[i]); err != nil {
			return "", err
		}
	}

	if len(files) > 0 {
		assembled = joinParts(assembled, formatFiles("Files", files))
	}
	if req.IncludeDirectory && len(directory.Files) > 0 {
		assembled = joinParts(assembled, formatFiles("Directory", directory.Files))
	}
	if strings.TrimSpace(req.PipedInput) != "" {
		assembled = joinParts(assembled, strings.TrimSpace(req.PipedInput))
	}

	return strings.TrimSpace(assembled), nil
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

func collectTemplateContents(cwd string, repo template.Repository, req domain.Request) ([]string, error) {
	var contents []string

	if indexTemplate, err := repo.Get("index"); err == nil {
		contents = append(contents, indexTemplate.Content)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	agentTemplates, err := collectAgentTemplates(cwd, req.TemplateNames)
	if err != nil {
		return nil, err
	}
	contents = append(contents, agentTemplates...)

	order := buildTemplateOrder(req.TemplateOrder, req.TemplateNames)
	for _, name := range order {
		if name == domain.BasePromptToken {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if strings.EqualFold(name, "agents.md") {
			continue
		}
		tmpl, err := repo.Get(name)
		if err != nil {
			return nil, err
		}
		contents = append(contents, tmpl.Content)
	}

	return contents, nil
}

func joinParts(left, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	switch {
	case left == "" && right == "":
		return ""
	case left == "":
		return right
	case right == "":
		return left
	default:
		return left + "\n\n" + right
	}
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
