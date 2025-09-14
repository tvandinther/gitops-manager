package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"sync"

	pb "github.com/tvandinther/gitops-manager/gen/go"
	"github.com/tvandinther/gitops-manager/pkg/client/request"
	"google.golang.org/protobuf/types/known/structpb"
)

type RequestContext struct {
	Context    context.Context
	Errors     []error
	summary    *JobSummary
	ProgressFn func(msg *pb.Progress)
	stream     pb.GitOps_UpdateManifestsClient
	receiveMsg sync.WaitGroup
	collectErr sync.WaitGroup
	errCh      chan error
}

type SendRequestOptions struct {
	ProgressFn func(msg *pb.Progress)
}

func WithContext(ctx context.Context) func(*RequestContext) {
	return func(r *RequestContext) {
		r.Context = ctx
	}
}

func WithProgressFn(fn func(msg *pb.Progress)) func(*RequestContext) {
	return func(r *RequestContext) {
		r.ProgressFn = fn
	}
}

func (c *Client) SendRequest(req *request.Request, opts ...func(*RequestContext)) (*RequestContext, error) {
	requestContext := &RequestContext{
		errCh:      make(chan error),
		Errors:     make([]error, 0),
		Context:    context.Background(),
		ProgressFn: PrintProgress,
	}

	for _, opt := range opts {
		opt(requestContext)
	}

	stream, err := c.grpcClient.UpdateManifests(requestContext.Context)
	if err != nil {
		log.Fatalf("error calling UpdateManifests procedure: %v", err)
	}
	requestContext.stream = stream

	requestContext.receiveMsg.Add(1)
	go requestContext.receiveMessages()
	requestContext.collectErr.Add(1)
	go requestContext.collectErrors()

	pbRequest, err := mapRequest(req)
	if err != nil {
		return nil, err
	}

	err = requestContext.stream.Send(pbRequest)

	if err != nil {
		log.Fatalf("error sending manifest request metadata: %v", err)
	}

	return requestContext, nil
}

func (r *RequestContext) receiveMessages() {
	defer r.receiveMsg.Done()

	summary := r.summary

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
		case <-r.stream.Context().Done():
			slog.Debug("context ended", "error", r.stream.Context().Err())
			return
		default:
		}

		msg, err := r.stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			r.errCh <- err

			return
		}

		switch m := msg.Response.(type) {
		case *pb.ManifestResponse_Progress:
			if r.ProgressFn != nil {
				r.ProgressFn(m.Progress)
			}
		case *pb.ManifestResponse_Error:
			r.errCh <- errors.New(m.Error.Message)
		case *pb.ManifestResponse_Summary:
			summary = &JobSummary{}
			summary.FromProto(m.Summary)
		}
	}
}

func (r *RequestContext) collectErrors() {
	defer r.collectErr.Done()

	for err := range r.errCh {
		if err != nil {
			r.Errors = append(r.Errors, err)
		}
	}
}

func (r *RequestContext) UploadDirectory(directory string) error {
	err := uploadDir(r.stream, directory)
	if err != nil {
		return fmt.Errorf("error uploading manifests: %w", err)
	}

	err = r.stream.CloseSend()
	if err != nil {
		return fmt.Errorf("error closing stream: %w", err)
	}

	return nil
}

func (r *RequestContext) Wait() *JobSummary {
	r.receiveMsg.Wait()
	close(r.errCh)
	r.collectErr.Wait()

	return r.summary
}

func mapRequest(req *request.Request) (*pb.ManifestRequest, error) {
	metadata, err := structpb.NewStruct(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create protobuf struct from metadata: %w", err)
	}

	sourceAttributes, err := structpb.NewStruct(req.Source.Metadata.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to create protobuf struct from source attributes: %w", err)
	}

	request := &pb.ManifestRequest{
		Content: &pb.ManifestRequest_Metadata{
			Metadata: &pb.UpdateManifestMetadata{
				ConfigRepository: &pb.Repository{
					Url: req.Repository.URL,
				},
				Environment:      req.Environment,
				AppName:          req.AppName,
				UpdateIdentifier: req.UpdateIdentifier,
				DryRun:           req.DryRun,
				AutoReview:       req.AutoReview,
				Source: &pb.RequestSource{
					Repository: &pb.Repository{
						Url: req.Source.Repository.URL,
					},
					Metadata: &pb.SourceMetadata{
						CommitSha:  req.Source.Metadata.CommitSHA,
						Actor:      req.Source.Metadata.Actor,
						Attributes: sourceAttributes.Fields,
					},
				},
				Metadata:   metadata.Fields,
				TotalFiles: int32(req.TotalFiles),
			},
		},
	}

	return request, nil
}
