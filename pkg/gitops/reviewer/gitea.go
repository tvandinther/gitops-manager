package reviewer

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Gitea struct {
	client       *gitea.Client
	mergeOptions *gitea.MergePullRequestOption
}

type GiteaMergeOptions struct {
	MergeStyle *gitea.MergeStyle
}

func (g *Gitea) CreateReview(ctx context.Context, req *gitops.Request, environmentBranches *gitops.EnvironmentBranches, sendMsg func(string)) (*gitops.CreateReviewResult, error) {
	owner, repo, err := getOwnerRepo(req.TargetRepository.URL)
	if err != nil {
		return nil, err
	}

	var pullRequest *gitea.PullRequest

	for pageIndex := 0; true; pageIndex++ {
		pullRequests, _, err := g.client.ListRepoPullRequests(owner, repo, gitea.ListPullRequestsOptions{
			State: gitea.StateOpen,
			ListOptions: gitea.ListOptions{
				Page:     pageIndex,
				PageSize: 100,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list repository pull requests: %w", err)
		}

		if len(pullRequests) == 0 {
			break
		}

		for _, pr := range pullRequests {
			slog.Debug("checking pull request for environment match", "baseName", pr.Base.Name, "headName", pr.Head.Name)
			if pr.Base.Name == environmentBranches.Trunk.Short() && pr.Head.Name == environmentBranches.Next.Short() {
				pullRequest = pr
				break
			}
		}
	}

	if pullRequest != nil {
		sendMsg("pull Request already exists")
		return &gitops.CreateReviewResult{
			Created:   true,
			URL:       pullRequest.URL,
			Completed: pullRequest.HasMerged,
		}, nil
	}

	pullRequest, response, err := g.client.CreatePullRequest(owner, repo, gitea.CreatePullRequestOption{
		Head:  environmentBranches.Next.Short(),
		Base:  environmentBranches.Trunk.Short(),
		Title: fmt.Sprintf("Promote %s [%s] to %s", req.AppName, req.UpdateIdentifier, req.Environment),
		Body: fmt.Sprintf(`<table>
  <tr>
    <td><strong>Target Environment</strong></td>
	<td>%s</td>
  </tr>
  <tr>
	<td><strong>Source Repository</strong></td>
	<td><a href="%s">%s/%s</a></td>
  </tr>
  <tr>
	<td><strong>Source Branch</strong></td>
	<td><a href="%s/tree/%s">%s</a></td>
  </tr>
  <tr>
	<td><strong>App Name</strong></td>
	<td>%s</td>
  </tr>
</table>`, req.Environment, req.Source.Repository.URL, owner, repo, req.Source.Repository.URL, req.UpdateIdentifier, req.UpdateIdentifier, req.AppName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}
	if response.StatusCode != 201 {
		sendMsg(fmt.Sprintf("received %d status code", response.StatusCode))
		return nil, fmt.Errorf("did not receieve 201 CREATED status code")
	}

	return &gitops.CreateReviewResult{
		Created:   true,
		URL:       pullRequest.URL,
		Completed: pullRequest.HasMerged,
	}, nil
}

func (g *Gitea) CompleteReview(ctx context.Context, req *gitops.Request, createReviewResult *gitops.CreateReviewResult, sendMsg func(string)) (bool, error) {
	owner, repo, err := getOwnerRepo(req.TargetRepository.URL)
	if err != nil {
		return false, err
	}

	pullRequestUrl, err := url.Parse(createReviewResult.URL)
	if err != nil {
		return false, fmt.Errorf("failed to parse pull request URL: %w", err)
	}
	pathSegments := strings.Split(pullRequestUrl.Path, "/")
	pullRequestId, err := strconv.ParseInt(pathSegments[len(pathSegments)-1], 10, 64)
	if err != nil {
		return false, fmt.Errorf("failed to parse ID from pull request URL: %w", err)
	}

	merged, response, err := g.client.MergePullRequest(owner, repo, pullRequestId, *g.mergeOptions)
	if err != nil {
		return false, fmt.Errorf("failed to merge pull request: %w", err)
	}
	if response.StatusCode != 200 {
		sendMsg(fmt.Sprintf("received %d status code", response.StatusCode))
		return false, fmt.Errorf("did not receieve 200 OK status code")
	}

	return merged, nil
}

func getOwnerRepo(repositoryUrl string) (string, string, error) {
	targetRepositoryUrl, err := url.Parse(repositoryUrl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse target repository URL: %w", err)
	}

	pathSegments := strings.Split(targetRepositoryUrl.Path, "/")
	ownerRepo := pathSegments[len(pathSegments)-2:]

	return ownerRepo[0], ownerRepo[1], nil
}
