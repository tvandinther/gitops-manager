package reviewer

import (
	"context"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Dummy struct {
	URL      string
	Complete bool
}

func (r *Dummy) CreateReview(ctx context.Context, req *gitops.Request, environmentBranches *gitops.EnvironmentBranches, sendMsg func(string)) (*gitops.CreateReviewResult, error) {
	result := &gitops.CreateReviewResult{
		Created:   true,
		Completed: false,
		URL:       r.URL,
	}

	return result, nil
}

func (r *Dummy) CompleteReview(ctx context.Context, req *gitops.Request, createReviewResult *gitops.CreateReviewResult, sendMsg func(string)) (bool, error) {
	return r.Complete, nil
}
