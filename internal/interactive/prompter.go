package interactive

import (
	"strings"

	"prompter-cli/internal/domain"
)

// UI describes the interactive prompt surface.
type UI interface {
	AskBasePrompt(defaultValue, note string) (string, error)
	SelectTemplates(templates []domain.Template, basePrompt string, preselected []string) ([]domain.Template, error)
}

// Prompter collects interactive input to build a request.
type Prompter struct {
	UI UI
}

// New builds an interactive prompter.
func New(ui UI) *Prompter {
	return &Prompter{UI: ui}
}

// Collect gathers base prompt and template selections.
func (p *Prompter) Collect(basePrompt string, templates []domain.Template, preselected []string, forcePrompt bool, note string) (domain.Request, error) {
	if p.UI == nil {
		return domain.Request{}, nil
	}

	prompt := strings.TrimSpace(basePrompt)
	if prompt == "" || forcePrompt {
		var err error
		prompt, err = p.UI.AskBasePrompt(prompt, note)
		if err != nil {
			return domain.Request{}, err
		}
	}

	selected := make([]domain.Template, 0)
	if len(templates) > 0 {
		var err error
		selected, err = p.UI.SelectTemplates(templates, prompt, preselected)
		if err != nil {
			return domain.Request{}, err
		}
	}
	names := make([]string, 0, len(selected))
	for _, tmpl := range selected {
		if tmpl.Name == "" {
			continue
		}
		names = append(names, tmpl.Name)
	}

	return domain.Request{
		BasePrompt:    prompt,
		TemplateNames: names,
	}, nil
}
