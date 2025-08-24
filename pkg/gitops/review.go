package gitops

import "context"

type CreateReviewResult struct {
	Created   bool
	URL       string
	Completed bool
}

type Reviewer interface {
	CreateReview(ctx context.Context) (CreateReviewResult, error)
	CompleteReview(ctx context.Context) (bool, error)
}
