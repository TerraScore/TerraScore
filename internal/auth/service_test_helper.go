package auth

// NewServiceForTest creates a Service with nil dependencies for validation-only tests.
// Validation checks return errors before any dependency is accessed.
func NewServiceForTest() *Service {
	return &Service{}
}
