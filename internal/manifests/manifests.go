package manifests

import "github.com/tvandinther/gitops-manager/pkg/progress"

type Repository = string

type UpdateManifestOptions struct {
	ConfigRepository *Repository
	Environment      string
	UpdateIdentifier string
	AppName          string
	Paths            Paths
	DryRun           bool
	AutoMerge        bool
	Source           *RequestSource
	TotalFiles       *int
}

type Paths struct {
	TempDir             string
	RepositoryDir       string
	UpdatedManifestsDir string
}

type RequestSource struct {
	Repository *Repository
	Metadata   *SourceMetadata
}

type SourceMetadata struct {
	CommitSHA     string
	PipelineActor string
	PipelineRunID string
}

type Response struct {
	Msg               string               `json:"msg"`
	Error             string               `json:"error"`
	PullRequest       *PullRequestResponse `json:"pullRequest"`
	Environment       *EnvironmentResponse `json:"environment"`
	UpdatedFilesCount int                  `json:"updatedFilesCount"`
	DryRun            bool                 `json:"dryRun"`
}

type PullRequestResponse struct {
	Created bool   `json:"created"`
	URL     string `json:"url"`
	Merged  bool   `json:"merged"`
}

type EnvironmentResponse struct {
	Repository *Repository `json:"repository"`
	Name       string      `json:"name"`
	RefName    string      `json:"refName"`
}

type ManifestUpdater struct {
	report *progress.Reporter
}
