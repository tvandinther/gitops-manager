package server

import (
	"context"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
	"github.com/tvandinther/gitops-manager/pkg/progress"
)

type Manager struct {
	RepositoryClient *gitops.Client
	report           *progress.Reporter
}

func NewManager(reporter *progress.Reporter, repositoryClient *gitops.Client) *Manager {
	return &Manager{
		RepositoryClient: repositoryClient,
		report:           reporter,
	}
}

type Repository = string

type RequestOptions struct {
	TargetRepository Repository
	Environment      string
	UpdateIdentifier string
	AppName          string
	Paths            Paths
	DryRun           bool
	AutoMerge        bool
	Source           *RequestSource
	TotalFiles       *int
}

type RequestSource struct {
	Repository *Repository
	Metadata   any
}

type Paths struct {
	TempDir             string
	RepositoryDir       string
	UpdatedManifestsDir string
}

type Response struct {
	Msg               string                     `json:"msg"`
	Error             string                     `json:"error"`
	ReviewResult      *gitops.CreateReviewResult `json:"pullRequest"`
	Environment       *EnvironmentResponse       `json:"environment"`
	UpdatedFilesCount int                        `json:"updatedFilesCount"`
	DryRun            bool                       `json:"dryRun"`
}

type EnvironmentResponse struct {
	Repository *Repository `json:"repository"`
	Name       string      `json:"name"`
	RefName    string      `json:"refName"`
}

func (m *Manager) ProcessRequest(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return nil, nil
}
