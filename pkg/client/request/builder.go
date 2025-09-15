package request

import (
	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Request struct {
	Repository       gitops.Repository
	Environment      string
	UpdateIdentifier string
	AppName          string
	DryRun           bool
	AutoReview       bool
	Source           RequestSource
	Metadata         map[string]any
	TotalFiles       int
}

type RequestSource struct {
	Repository gitops.Repository
	Metadata   RequestSourceMetadata
}

type RequestSourceMetadata struct {
	CommitSHA  string
	Actor      string
	Attributes map[string]any
}

func New(opts ...func(*Request)) *Request {
	request := &Request{}

	for _, opt := range opts {
		opt(request)
	}

	return request
}

func WithRepository(repo gitops.Repository) func(*Request) {
	return func(r *Request) {
		r.Repository = repo
	}
}

func WithEnvironment(env string) func(*Request) {
	return func(r *Request) {
		r.Environment = env
	}
}

func WithUpdateIdentifier(identifier string) func(*Request) {
	return func(r *Request) {
		r.UpdateIdentifier = identifier
	}
}

func WithAppName(name string) func(*Request) {
	return func(r *Request) {
		r.AppName = name
	}
}

func WithDryRun(dryRun bool) func(*Request) {
	return func(r *Request) {
		r.DryRun = dryRun
	}
}

func WithAutoReview(autoReview bool) func(*Request) {
	return func(r *Request) {
		r.AutoReview = autoReview
	}
}

func WithSource(source RequestSource) func(*Request) {
	return func(r *Request) {
		r.Source = source
	}
}

func WithMetadata(metadata map[string]any) func(*Request) {
	return func(r *Request) {
		r.Metadata = metadata
	}
}

func WithTotalFiles(totalFiles int) func(*Request) {
	return func(r *Request) {
		r.TotalFiles = totalFiles
	}
}
