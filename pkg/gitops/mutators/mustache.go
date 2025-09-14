package mutators

import (
	"context"
	"io"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

// This mutator performs Mustache templating
type Mustache struct {
	GetData func(request *gitops.Request) (map[string]interface{}, error)
}

func (m *Mustache) GetTitle() string {
	return "Mustache template"
}

func (m *Mustache) MutateFile(ctx context.Context, request *gitops.Request, inputFile io.Reader, outputFile io.Writer, sendMsg func(string)) error {
}
