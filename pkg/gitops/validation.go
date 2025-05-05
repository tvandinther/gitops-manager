package gitops

type ValidationResult struct {
	IsValid bool
	Errors  []error
}

type Validator interface {
	ValidateDir(path string) (ValidationResult, error)
}
