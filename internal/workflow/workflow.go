package workflow

import (
	"errors"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/output"
	"prompter-cli/internal/template"
)

// Service holds shared dependencies for workflows.
type Service struct {
	Generator *Generator
	Output    *output.Handler
}

// New builds a new workflow service container.
func New(repo template.Repository, out *output.Handler) *Service {
	return &Service{
		Generator: NewGenerator(repo),
		Output:    out,
	}
}

// Generate builds the prompt and routes it to the configured target if an output handler is available.
func (s *Service) Generate(req domain.Request, cfg domain.Config) (string, error) {
	if s.Generator == nil {
		return "", errors.New("generator is required")
	}
	content, err := s.Generator.Run(req, cfg)
	if err != nil {
		return "", err
	}
	if s.Output != nil {
		if err := s.Output.Write(req, content, cfg); err != nil {
			return "", err
		}
	}
	return content, nil
}
