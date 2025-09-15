package mutators

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/cbroglie/mustache"
	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

// This mutator performs Mustache templating
type Mustache struct {
	GetData func(request *gitops.Request) (any, error)
}

func (m *Mustache) GetTitle() string {
	return "Mustache template"
}

func (m *Mustache) MutateFile(ctx context.Context, request *gitops.Request, inputFile io.Reader, outputFile io.Writer, sendMsg func(string)) error {
	inputData, err := io.ReadAll(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	data, err := m.GetData(request)
	if err != nil {
		return fmt.Errorf("failed to get template data: %w", err)
	}
	templated, err := mustache.Render(string(inputData), data)
	fmt.Println(templated)

	bytesWritten, err := io.WriteString(outputFile, templated)
	slog.Debug("written to output file", "bytesWritten", bytesWritten)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
