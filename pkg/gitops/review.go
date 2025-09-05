package gitops

import "context"

type CreateReviewResult struct {
	Created   bool
	URL       string
	Completed bool
}

type Reviewer interface {
	CreateReview(ctx context.Context, req *Request, environmentBranches *EnvironmentBranches, sendMsg func(string)) (*CreateReviewResult, error)
	CompleteReview(ctx context.Context, req *Request, createReviewResult *CreateReviewResult, sendMsg func(string)) (bool, error)
}
