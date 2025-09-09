package client

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/tvandinther/gitops-manager/gen/go"
)

type Client struct {
	serverHost     string
	requestOptions *RequestOptions
}

type RequestOptions struct {
	TargetRepository  *string
	Environment       *string
	AppName           *string
	UpdateIdentifier  *string
	DryRun            *bool
	AutoReview        *bool
	ManifestDirectory string
	SourceRepository  *string
	CommitSHA         *string
	Actor             *string
	SourceAttributes  *string
}

func New() *Client {
	client := &Client{}

	client.requestOptions = &RequestOptions{
		TargetRepository: flag.String("target-repository", "", "Name of the target configuration repository"),
		Environment:      flag.String("env", "", "Target environment"),
		AppName:          flag.String("app", "", "Application name"),
		UpdateIdentifier: flag.String("update-id", "", "Update identifier (e.g., git branch)"),
		DryRun:           flag.Bool("dry-run", true, "Enable dry-run mode"),
		AutoReview:       flag.Bool("auto-review", false, "Enable automatic completion of reviews"),
		SourceRepository: flag.String("source-repository", "", "Source repository"),
		CommitSHA:        flag.String("commit-sha", "", "Commit SHA"),
		Actor:            flag.String("actor", "", "Actor"),
		SourceAttributes: flag.String("source-attributes", "", "Source attributes"),
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		log.Fatal("usage: program [options] <manifests-directory> <gitops-server-url>")
	}
	client.requestOptions.ManifestDirectory = args[0]
	client.serverHost = args[1]

	return client
}

func (c *Client) Run() {
	secure := getEnvBool("GITOPS_SECURE", false)

	var creds credentials.TransportCredentials
	if secure {
		creds = credentials.NewClientTLSFromCert(nil, "")
	} else {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(
		c.serverHost,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             2 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		log.Fatalf("could not create client: %v", err)
	}
	defer conn.Close()

	client := pb.NewGitOpsClient(conn)

	timeout := 30 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var sourceAttributesMap map[string]any
	err = json.Unmarshal([]byte(*c.requestOptions.SourceAttributes), &sourceAttributesMap)
	if err != nil {
		log.Fatalf("failed to unmarshal JSON from source attributes: %v", err)
	}

	protoStruct, err := structpb.NewStruct(sourceAttributesMap)
	if err != nil {
		log.Fatalf("failed to create protobuf struct from source attributes: %v", err)
	}

	absDirPath, err := filepath.Abs(c.requestOptions.ManifestDirectory)
	if err != nil {
		log.Fatalf("error parsing manifests directory: %s", c.requestOptions.ManifestDirectory)
	}

	totalFiles, err := countFiles(absDirPath)
	if err != nil {
		log.Fatalf("error while counting files in manifest directory")
	}

	req := &pb.ManifestRequest{
		Content: &pb.ManifestRequest_Metadata{
			Metadata: &pb.UpdateManifestMetadata{
				ConfigRepository: &pb.Repository{
					Url: *c.requestOptions.TargetRepository,
				},
				Environment:      *c.requestOptions.Environment,
				AppName:          *c.requestOptions.AppName,
				UpdateIdentifier: *c.requestOptions.UpdateIdentifier,
				DryRun:           *c.requestOptions.DryRun,
				AutoReview:       *c.requestOptions.AutoReview,
				Source: &pb.RequestSource{
					Repository: &pb.Repository{
						Url: *c.requestOptions.SourceRepository,
					},
					Metadata: &pb.SourceMetadata{
						CommitSha:  *c.requestOptions.CommitSHA,
						Actor:      *c.requestOptions.Actor,
						Attributes: protoStruct.Fields,
					},
				},
				TotalFiles: int32(totalFiles),
			},
		},
	}
	PrettyPrintManifestRequest(req)

	notified := false
	timer := time.AfterFunc(2*time.Second, func() {
		fmt.Println("Connecting to server... This may take a moment due to a cold start.")
		notified = true
	})

	stream, err := client.UpdateManifests(ctx)
	if err != nil {
		log.Fatalf("error calling UpdateManifests procedure: %v", err)
	}
	timer.Stop()
	if notified {
		fmt.Println("Connection established.")
	}

	errCh := make(chan error)
	errors := make([]error, 0)
	var (
		receiveMsg sync.WaitGroup
		collectErr sync.WaitGroup
	)

	receiveMsg.Add(1)
	go receiveMessages(stream, errCh, &receiveMsg)
	collectErr.Add(1)
	go collectErrors(&errors, errCh, &collectErr)

	err = stream.Send(req)

	if err != nil {
		log.Fatalf("error sending manifest request metadata: %v", err)
	}

	err = uploadDir(stream, absDirPath)
	if err != nil {
		log.Fatalf("error uploading manifests: %v", err)
	}

	stream.CloseSend()
	receiveMsg.Wait()

	close(errCh)
	collectErr.Wait()

	for _, err := range errors {
		fmt.Printf("[ERROR] %v\n", err)
	}

	if len(errors) > 0 {
		os.Exit(1)
	}
}

func collectErrors(errors *[]error, errCh <-chan error, wg *sync.WaitGroup) {
	defer wg.Done()

	for err := range errCh {
		if err != nil {
			*errors = append(*errors, err)
		}
	}
}

func getEnvBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		log.Fatalf("invalid boolean value for %s: %v", key, err)
	}
	return val
}

func receiveMessages(stream grpc.BidiStreamingClient[pb.ManifestRequest, pb.ManifestResponse], errCh chan error, wg *sync.WaitGroup) {
	defer wg.Done()

	var (
		summary *JobSummary
	)

	defer func() {
		if summary != nil {
			jsonSummary, err := json.Marshal(summary)
			if err != nil {
				slog.Error("failed to marshal job summary into JSON", "error", err)
			}
			fmt.Println()
			PrettyPrintJSONBlock("Result", jsonSummary)
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			slog.Debug("context ended", "error", stream.Context().Err())
			return
		default:
		}

		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			errCh <- err

			return
		}

		switch m := msg.Response.(type) {
		case *pb.ManifestResponse_Progress:
			PrintProgress(m.Progress)
		case *pb.ManifestResponse_Error:
			errCh <- errors.New(m.Error.Message)
		case *pb.ManifestResponse_Summary:
			summary = &JobSummary{}
			summary.FromProto(m.Summary)
		}
	}
}

func countFiles(directory string) (int, error) {
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
