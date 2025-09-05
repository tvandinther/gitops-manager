package validators

import (
	"context"
	"fmt"
	"io"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type EmptyFile struct{}

func (e *EmptyFile) GetTitle() string {
	return "Empty File"
}

func (e *EmptyFile) ValidateFile(ctx context.Context, file io.Reader, sendMsg func(string)) (*gitops.ValidationResult, error) {
	result := &gitops.ValidationResult{
		IsValid: false,
		Errors:  make([]error, 0),
	}

	buf := make([]byte, 1)

	n, err := file.Read(buf)

	if n > 0 {
		result.IsValid = true
		return result, nil
	}

	if err == io.EOF {
		result.Errors = append(result.Errors, fmt.Errorf("file is empty: %w", err))
		return result, nil
	}

	return result, err
}
