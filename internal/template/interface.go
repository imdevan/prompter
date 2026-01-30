package template

import "prompter-cli/internal/domain"

// Repository describes template storage behavior.
type Repository interface {
	List() ([]domain.Template, error)
	Get(name string) (domain.Template, error)
	Save(template domain.Template) error
}
