package testutil

import "prompter-cli/internal/domain"

// FakeUI implements the interactive.UI interface for tests.
type FakeUI struct {
	BasePrompt string
	Selected   []domain.Template
	Err        error
}

func (f FakeUI) AskBasePrompt(defaultValue, note string) (string, error) {
	if f.Err != nil {
		return "", f.Err
	}
	if f.BasePrompt != "" {
		return f.BasePrompt, nil
	}
	return defaultValue, nil
}

func (f FakeUI) SelectTemplates(templates []domain.Template, basePrompt string, preselected []string) ([]domain.Template, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Selected != nil {
		return f.Selected, nil
	}
	return nil, nil
}
