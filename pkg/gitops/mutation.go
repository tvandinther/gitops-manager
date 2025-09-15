package gitops

import (
	"context"
	"io"
)

type Mutator interface {
	GetTitle() string
	MutateFile(ctx context.Context, request *Request, inputFile io.Reader, outputFile io.Writer, sendMsg func(string)) error
}
