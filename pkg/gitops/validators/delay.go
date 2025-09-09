package validators

import (
	"context"
	"io"
	"time"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Delay struct {
	Duration time.Duration
}

func (d *Delay) GetTitle() string {
	return "Delay"
}

func (d *Delay) ValidateFile(ctx context.Context, file io.Reader, sendMsg func(string)) (*gitops.ValidationResult, error) {
	time.Sleep(d.Duration)

	result := &gitops.ValidationResult{
		IsValid: true,
		Errors:  make([]error, 0),
	}

	return result, nil
}
