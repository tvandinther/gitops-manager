package gitops

import (
	"context"
	"io"
)

type ValidationResult struct {
	IsValid bool
	Errors  []error
}

type Validator interface {
	GetTitle() string
	ValidateFile(ctx context.Context, file io.Reader, sendMsg func(string)) (*ValidationResult, error)
}
