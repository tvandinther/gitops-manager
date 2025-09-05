package util

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type TempFS struct {
	Root        string
	Directories []string
}

func NewTempFS() (*TempFS, error) {
	tempDir, err := os.MkdirTemp("", "gitops-manager-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary app root directory: %w", err)
	}

	slog.Debug("made new temporary root directory", "path", tempDir)

	tempFS := &TempFS{
		Root:        tempDir,
		Directories: make([]string, 0),
	}

	return tempFS, nil
}

func (t *TempFS) Clear() error {
	return os.RemoveAll(t.Root)
}

func (t *TempFS) Mkdir(paths ...string) (string, error) {
	dirPath := filepath.Join(paths...)
	if !strings.HasPrefix(dirPath, t.Root) {
		dirPath = filepath.Join(t.Root, dirPath)
	}
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to make directory %s: %w", dirPath, err)
	}

	t.Directories = append(t.Directories, dirPath)

	slog.Debug("made temporary directory", "path", dirPath)

	return dirPath, nil
}

func Map[T any, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, e := range in {
		out[i] = f(e)
	}

	return out
}
