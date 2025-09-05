package progress

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	pb "github.com/tvandinther/gitops-manager/gen/go"
	"google.golang.org/grpc"
)

type Reporter struct {
	stream       grpc.BidiStreamingServer[pb.ManifestRequest, pb.ManifestResponse]
	progressChan chan *pb.Progress
}

func NewReporter(stream grpc.BidiStreamingServer[pb.ManifestRequest, pb.ManifestResponse]) *Reporter {
	return &Reporter{
		stream:       stream,
		progressChan: make(chan *pb.Progress),
	}
}

func (p *Reporter) Run(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case <-ctx.Done():
			return
		case prog, ok := <-p.progressChan:
			if !ok {
				wg.Done()
				return
			}
			err := p.stream.Send(&pb.ManifestResponse{
				Response: &pb.ManifestResponse_Progress{
					Progress: prog,
				},
			})

			if err != nil {
				slog.Error("failed to send progress update", "update", prog.Status, "error", err)
			}
		}
	}
}

func (p *Reporter) Close() {
	close(p.progressChan)
}

func (p *Reporter) Heading(s string) {
	p.progressChan <- &pb.Progress{
		Kind:   pb.ProgressKind_HEADING,
		Status: s,
	}
}

func (p *Reporter) Progress(s string, args ...any) {
	p.progressChan <- &pb.Progress{
		Kind:   pb.ProgressKind_PROGRESS,
		Status: fmt.Sprintf(s, args...),
	}
}

func (p *Reporter) BasicProgress(s string) {
	p.Progress("%s", s)
}

func (p *Reporter) Success(s string, args ...any) {
	p.progressChan <- &pb.Progress{
		Kind:   pb.ProgressKind_SUCCESS,
		Status: fmt.Sprintf(s, args...),
	}
}

func (p *Reporter) Failure(s string, args ...any) {
	p.progressChan <- &pb.Progress{
		Kind:   pb.ProgressKind_FAILURE,
		Status: fmt.Sprintf(s, args...),
	}
}

type Result struct {
	Success string
	Failure string
}

// Sends a failure progress if err is not nil, else a success progress
func (p *Reporter) Result(err error, result Result) {
	if err != nil {
		p.Failure("%s", result.Failure)
	} else {
		p.Success("%s", result.Success)
	}
}
