package gitops

type Repository struct {
	URL string
}

type Request struct {
	TargetRepository Repository
	Environment      string
	UpdateIdentifier string
	AppName          string
	Paths            Paths
	DryRun           bool
	AutoReview       bool
	Source           *RequestSource
	TotalFiles       *int
	Metadata         map[string]any
}

type RequestSource struct {
	Repository *Repository
	Metadata   *RequestSourceMetadata
}

type RequestSourceMetadata struct {
	CommitSHA  string
	Actor      string
	Attributes map[string]any
}

type Paths struct {
	TempDir             string
	RepositoryDir       string
	UpdatedManifestsDir string
}

type Response struct {
	Msg               string               `json:"msg"`
	Error             string               `json:"error"`
	ReviewResult      *CreateReviewResult  `json:"pullRequest"`
	Environment       *EnvironmentResponse `json:"environment"`
	UpdatedFilesCount int                  `json:"updatedFilesCount"`
	DryRun            bool                 `json:"dryRun"`
}

type EnvironmentResponse struct {
	Repository *Repository `json:"repository"`
	Name       string      `json:"name"`
	RefName    string      `json:"refName"`
}
