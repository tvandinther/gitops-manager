package reviewer

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Gitlab struct {
	Client       *gitlab.Client
	MergeOptions *GitlabMergeOptions
}

type GitlabMergeOptions struct {
	Squash        bool
	CommitMessage string
	DeleteBranch  bool
}

func (g *Gitlab) CreateReview(ctx context.Context, req *gitops.Request, target *gitops.Target, sendMsg func(string)) (*gitops.CreateReviewResult, error) {
	projectId, err := g.getProjectId(target.Repository.URL)
	if err != nil {
		return nil, err
	}

	var mergeRequest *gitlab.BasicMergeRequest

	slog.Info("listing merge requests to find existing", "projectId", projectId)
	mergeRequests, _, err := g.Client.MergeRequests.ListProjectMergeRequests(projectId, &gitlab.ListProjectMergeRequestsOptions{
		State:        gitlab.Ptr("opened"),
		SourceBranch: gitlab.Ptr(target.Branch.Source),
		TargetBranch: gitlab.Ptr(target.Branch.Target),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list repository merge requests: %w", err)
	}

	if len(mergeRequests) != 0 {
		mergeRequest = mergeRequests[0]
	}

	if mergeRequest != nil {
		sendMsg("merge Request already exists")
		return &gitops.CreateReviewResult{
			Created:   true,
			URL:       mergeRequest.WebURL,
			Completed: mergeRequest.State == "merged",
		}, nil
	}

	mr, response, err := g.Client.MergeRequests.CreateMergeRequest(projectId, &gitlab.CreateMergeRequestOptions{
		TargetBranch: gitlab.Ptr(target.Branch.Target),
		SourceBranch: gitlab.Ptr(target.Branch.Source),
		Title:        gitlab.Ptr(fmt.Sprintf("Promote %s [%s] to %s", req.AppName, req.UpdateIdentifier, req.Environment)),
		Description: gitlab.Ptr(fmt.Sprintf(`<table>
  <tr>
    <td><strong>Target Environment</strong></td>
	<td>%s</td>
  </tr>
  <tr>
	<td><strong>Source Repository</strong></td>
	<td><a href="%s">%s</a></td>
  </tr>
  <tr>
	<td><strong>Source Branch</strong></td>
	<td><a href="%s/tree/%s">%s</a></td>
  </tr>
  <tr>
	<td><strong>App Name</strong></td>
	<td>%s</td>
  </tr>
</table>`, req.Environment, req.Source.Repository.URL, projectId, req.Source.Repository.URL, req.UpdateIdentifier, req.UpdateIdentifier, req.AppName),
		)})
	if err != nil {
		return nil, fmt.Errorf("failed to create merge request: %w", err)
	}
	if response.StatusCode != 201 {
		sendMsg(fmt.Sprintf("received %d status code", response.StatusCode))
		return nil, fmt.Errorf("did not receieve 201 CREATED status code")
	}

	result := &gitops.CreateReviewResult{
		Created:   true,
		URL:       mr.WebURL,
		Completed: mr.State == "merged",
	}
	slog.Info("created merge request", "result", result)

	return result, nil
}

func (g *Gitlab) CompleteReview(ctx context.Context, req *gitops.Request, createReviewResult *gitops.CreateReviewResult, sendMsg func(string)) (bool, error) {
	projectId, mergeRequestId, err := g.getProjectIdAndMergeRequestId(createReviewResult.URL)
	if err != nil {
		return false, err
	}

	acceptOptions := &gitlab.AcceptMergeRequestOptions{
		AutoMerge:                gitlab.Ptr(true),
		Squash:                   gitlab.Ptr(g.MergeOptions.Squash),
		ShouldRemoveSourceBranch: gitlab.Ptr(g.MergeOptions.DeleteBranch),
	}
	if g.MergeOptions.Squash {
		acceptOptions.SquashCommitMessage = gitlab.Ptr(g.MergeOptions.CommitMessage)
	} else {
		acceptOptions.MergeCommitMessage = gitlab.Ptr(g.MergeOptions.CommitMessage)
	}

	sendMsg("merging merge request")
	retries := 0
	retryLimit := 1
Merge:
	mergeRequest, response, err := g.Client.MergeRequests.AcceptMergeRequest(projectId, mergeRequestId, acceptOptions)
	if err != nil {
		return false, fmt.Errorf("failed to merge merge request: %w", err)
	}
	if response.StatusCode != 200 {
		if response.StatusCode == 405 && retries < retryLimit {
			time.Sleep(1 * time.Second)
			retries++
			goto Merge
		}
		sendMsg(fmt.Sprintf("received %d status code", response.StatusCode))
		return false, fmt.Errorf("did not receieve 200 OK status code")
	}
	merged := mergeRequest.State == "merged"
	if !merged {
		return false, fmt.Errorf("failed to merge for an unknown reason")
	}
	sendMsg("merge request merged")

	return merged, nil
}

func (g *Gitlab) getProjectId(repositoryUrl string) (string, error) {
	targetRepositoryUrl, err := url.Parse(repositoryUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse target repository URL: %w", err)
	}

	projectId := strings.TrimSuffix(targetRepositoryUrl.Path, ".git")

	return projectId, nil
}

func (g *Gitlab) getProjectIdAndMergeRequestId(mergeRequestUrl string) (string, int, error) {
	targetRepositoryUrl, err := url.Parse(mergeRequestUrl)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse target repository URL: %w", err)
	}

	splitPath := strings.Split(targetRepositoryUrl.Path, "/-/merge_requests/")
	projectId := splitPath[0]
	mergeRequestId, err := strconv.ParseInt(splitPath[1], 10, 32)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse merge request id into integer: %w", err)
	}

	return projectId, int(mergeRequestId), nil
}
