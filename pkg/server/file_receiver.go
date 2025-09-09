package server

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	pb "github.com/tvandinther/gitops-manager/gen/go"
	"github.com/tvandinther/gitops-manager/pkg/progress"
)

type FileReceiver struct {
	report               *progress.Reporter
	progress             *progress.ProcessReporter
	buffers              map[string]*bytes.Buffer
	destinationDirectory string
	startedReceiving     bool
	filesReceived        int
	totalFiles           int
}

func newFileReceiver(destinationDirectory string, totalFiles int, reporter *progress.Reporter) *FileReceiver {
	progress := reporter.NewProcess(&progress.ProcessReporterOptions{
		ReportPeriod:   2 * time.Second,
		TotalFileCount: totalFiles,
		Template: progress.ProcessTemplate{
			PresentAction: "receiving",
			PastAction:    "received",
			Subject:       "files",
		},
	})

	return &FileReceiver{
		report:               reporter,
		progress:             progress,
		buffers:              make(map[string]*bytes.Buffer),
		destinationDirectory: destinationDirectory,
		startedReceiving:     false,
		totalFiles:           totalFiles,
	}
}

func (r *FileReceiver) receiveFileChunk(ctx context.Context, chunk *pb.FileChunk) error {
	if !r.startedReceiving {
		r.startedReceiving = true
		r.progress.Heading("Receiving files")
		r.progress.Start(ctx)
	}
	slog.Debug("received file chunk", "filename", chunk.Filename, "isLast", chunk.IsLastChunk)
	buffer, exists := r.buffers[chunk.Filename]
	if !exists {
		buffer = &bytes.Buffer{}
		r.buffers[chunk.Filename] = buffer
	}
	buffer.Write(chunk.Content)
	if chunk.IsLastChunk {
		slog.Debug("Received full file", "name", chunk.Filename, "size", r.buffers[chunk.Filename].Len())
		absoluteFilename := filepath.Join(r.destinationDirectory, chunk.Filename)
		err := os.MkdirAll(filepath.Dir(absoluteFilename), os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to make parent directory: %w", err)
		}
		slog.Debug("writing file", "filename", chunk.Filename, "absoluteFilename", absoluteFilename)

		r.filesReceived++
		r.progress.Increment(1)

		err = os.WriteFile(absoluteFilename, buffer.Bytes(), os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		delete(r.buffers, chunk.Filename)
	}

	return nil
}

func (r *FileReceiver) done() error {
	r.progress.Done()

	if !r.startedReceiving {
		r.progress.Failure("No files received")
		return fmt.Errorf("no files received")
	}

	if len(r.buffers) > 1 {
		r.progress.Failure("Partially received %d files", len(r.buffers))
		return fmt.Errorf("partially received %d files", len(r.buffers))
	}

	if r.filesReceived != r.totalFiles {
		r.progress.Failure("Received %d/%d files", r.filesReceived, r.totalFiles)
		return fmt.Errorf("received %d/%d files", r.filesReceived, r.totalFiles)
	}

	r.progress.Success("Received all files")
	return nil
}

func (r *FileReceiver) getFilesReceived() int {
	return r.filesReceived
}
