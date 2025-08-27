package gitops

import (
	"context"
	"io/fs"
)

type ValidationResult struct {
	IsValid bool
	Errors  []error
}

type Validator interface {
	GetTitle() string
	Validate(ctx context.Context, dir fs.FS, error func(e error), done func(result ValidationResult), sendMsg func(string))
}
