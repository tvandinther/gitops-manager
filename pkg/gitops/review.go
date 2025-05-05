package gitops

import "context"

type CreateReviewResult struct {
	Created bool
	URL     string
}

type Reviewer interface {
	CreateReview(ctx context.Context) (CreateReviewResult, error)
	CompleteReview(ctx context.Context) (bool, error)
}
