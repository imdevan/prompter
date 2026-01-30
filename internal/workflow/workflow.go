package workflow

// Service holds shared dependencies for workflows.
type Service struct{}

// New builds a new workflow service container.
func New() *Service {
	return &Service{}
}
