package client

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/tvandinther/gitops-manager/gen/go"
	"google.golang.org/grpc"
)

func CountFiles(directory string) (int, error) {
	count := 0

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			count++
		}

		return nil
	})

	if err != nil {
		return count, fmt.Errorf("error walking directory: %w", err)
	}

	return count, nil
}

func uploadDir(stream grpc.BidiStreamingClient[pb.ManifestRequest, pb.ManifestResponse], directory string) error {
	fileInfo, err := os.Stat(directory)

	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", directory)
	}

	return filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		return uploadFile(directory, path, stream)
	})
}

func uploadFile(root, path string, stream grpc.BidiStreamingClient[pb.ManifestRequest, pb.ManifestResponse]) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", path, err)
	}

	truncatedPath, valid := strings.CutPrefix(path, root+string(os.PathSeparator))
	if !valid {
		return errors.New("invalid root path")
	}

	fileChunk := &pb.FileChunk{
		Filename:    truncatedPath,
		Content:     data,
		IsLastChunk: true,
	}

	slog.Debug("sending file chunk", "filename", fileChunk.Filename, "isLast", fileChunk.IsLastChunk)
	err = stream.Send(&pb.ManifestRequest{
		Content: &pb.ManifestRequest_File{
			File: fileChunk,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send %q: %w", path, err)
	}

	return nil
}
