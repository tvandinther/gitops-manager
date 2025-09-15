package mutators

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

// This mutator ensures files end with a new line character
type NewLineEOF struct{}

func (_ *NewLineEOF) GetTitle() string {
	return "New Line EOF"
}

func (_ *NewLineEOF) MutateFile(ctx context.Context, request *gitops.Request, inputFile io.Reader, outputFile io.Writer, sendMsg func(string)) error {
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	if buf.Len() > 0 {
		data := buf.Bytes()
		lastByte := data[buf.Len()-1]
		if lastByte == '\n' {
			return nil
		} else {
			data = append(data, byte('\n'))
		}
		_, err := outputFile.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	}

	return nil
}
