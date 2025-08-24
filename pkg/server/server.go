package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/tvandinther/gitops-manager/gen/go"
	"github.com/tvandinther/gitops-manager/internal/health"
	"github.com/tvandinther/gitops-manager/internal/util"
	"github.com/tvandinther/gitops-manager/pkg/progress"

	"log/slog"

	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedGitOpsServer
}

func New() *Server {
	return &Server{}
}

func (s *Server) WithDefaultLogger() *Server {
	var logLevel slog.Level
	level, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = slog.LevelInfo
	} else {
		err := logLevel.UnmarshalText([]byte(level))
		if err != nil {
			panic("Invalid value set for 'LOG_LEVEL'. Use a valid level string for unmarshalling with the log/slog package.")
		}
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("logger initialised", "logLevel", logLevel)

	return s
}

func (s *Server) Run() error {
	listenAddr := ":50051"
	if val, ok := os.LookupEnv("PORT"); ok {
		listenAddr = ":" + val
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterGitOpsServer(grpcServer, s)

	http.HandleFunc("/health", health.Handler)
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("gRPC server listening", "port", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("gRPC server failed", "error", err)
		return err
	}

	return nil
}

func (s *Server) UpdateManifests(stream grpc.BidiStreamingServer[pb.ManifestRequest, pb.ManifestResponse]) error {
	slog.Info("opened stream")

	ctx := stream.Context()
	var wg sync.WaitGroup

	reporter := progress.NewReporter(stream)
	wg.Add(1)
	go reporter.Run(ctx, &wg)

	tempfs, err := util.NewTempFS()
	if err != nil {
		return fmt.Errorf("failed creating a new temporary filesystem: %w", err)
	}
	defer tempfs.Clear()

	repoDir, err := tempfs.Mkdir("repository")
	if err != nil {
		return fmt.Errorf("failed to make repository directory: %w", err)
	}

	updatedManifestsPath, err := tempfs.Mkdir("upload")
	if err != nil {
		return fmt.Errorf("failed to make upload directory: %w", err)
	}

	var (
		fileBuffers               = make(map[string]*bytes.Buffer)
		metadataRecieved          = false
		receivedFileCount    int  = 0
		receivedFileCountPtr *int = &receivedFileCount
		opts                      = &RequestOptions{
			Paths: Paths{
				TempDir:             tempfs.Root,
				RepositoryDir:       repoDir,
				UpdatedManifestsDir: updatedManifestsPath,
			},
			TotalFiles: receivedFileCountPtr,
		}
	)

	reporter.Heading("Receiving data")
	for {
		select {
		case <-ctx.Done():
			slog.Info("context ended")
			return ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				reporter.Success("Data received")
				goto respond
			}
			return fmt.Errorf("failed to receive message: %w", err)
		}
		switch m := msg.Content.(type) {
		case *pb.ManifestRequest_Metadata:
			reporter.Progress("received request metadata")
			if metadataRecieved {
				return errors.New("request metadata already recieved")
			}
			metadataRecieved = true
			assignRequestMetadataToOptions(m.Metadata, opts)

		case *pb.ManifestRequest_File:
			chunk := m.File
			slog.Debug("received file chunk", "filename", chunk.Filename, "isLast", chunk.IsLastChunk)
			buffer, exists := fileBuffers[chunk.Filename]
			if !exists {
				buffer = &bytes.Buffer{}
				fileBuffers[chunk.Filename] = buffer
			}
			buffer.Write(chunk.Content)
			if chunk.IsLastChunk {
				slog.Debug("Received full file", "name", chunk.Filename, "size", fileBuffers[chunk.Filename].Len())
				absoluteFilename := filepath.Join(opts.Paths.UpdatedManifestsDir, chunk.Filename)
				err = os.MkdirAll(filepath.Dir(absoluteFilename), os.ModePerm)
				if err != nil {
					return fmt.Errorf("failed to make parent directory: %w", err)
				}
				slog.Debug("writing file", "filename", chunk.Filename, "absoluteFilename", absoluteFilename)

				receivedFileCount++
				reporter.Progress("received file %s", chunk.Filename)

				err = os.WriteFile(absoluteFilename, buffer.Bytes(), os.ModePerm)
				if err != nil {
					return fmt.Errorf("failed to write file: %w", err)
				}
				delete(fileBuffers, chunk.Filename)
			}
		}
	}

respond:
	if err := ctx.Err(); err != nil {
		log.Printf("Context error before processing: %v", err)
		return err
	}

	manager := NewManager(reporter, nil)

	response, err := manager.ProcessRequest(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to process manifests request: %w", err)
	}

	if response.Error != "" {
		slog.Info("sending error response", "error", response.Error)
		err = stream.Send(&pb.ManifestResponse{
			Response: &pb.ManifestResponse_Error{
				Error: &pb.Error{
					Message: response.Error,
				},
			},
		})
	} else {
		slog.Info("sending summary", "message", response.Msg)
		err = stream.Send(&pb.ManifestResponse{
			Response: &pb.ManifestResponse_Summary{
				Summary: &pb.Summary{
					Message:           response.Msg,
					UpdatedFilesCount: int32(response.UpdatedFilesCount),
					DryRun:            response.DryRun,
					Review: &pb.ReviewSummary{
						Created: response.ReviewResult.Created,
						Url:     response.ReviewResult.URL,
						Merged:  response.ReviewResult.Completed,
					},
					Environment: &pb.EnvironmentSummary{
						Repository: &pb.Repository{
							Url: *response.Environment.Repository,
						},
						Name:    response.Environment.Name,
						RefName: response.Environment.RefName,
					},
				},
			},
		})
	}
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	reporter.Close()
	wg.Wait()

	return nil
}

func assignRequestMetadataToOptions(metadata *pb.UpdateManifestMetadata, opts *RequestOptions) {
	opts.TargetRepository = metadata.ConfigRepository.Url
	opts.Environment = metadata.Environment
	opts.UpdateIdentifier = metadata.UpdateIdentifier
	opts.AppName = metadata.AppName
	opts.DryRun = metadata.DryRun
	opts.AutoMerge = metadata.AutoMerge
	opts.Source = &RequestSource{
		Repository: &metadata.Source.Repository.Url,
		Metadata: struct {
			CommitSHA     string
			PipelineActor string
			PipelineRunID string
		}{
			CommitSHA:     metadata.Source.Metadata.CommitSha,
			PipelineActor: metadata.Source.Metadata.PipelineActor,
			PipelineRunID: metadata.Source.Metadata.PipelineRunId,
		},
	}
}
