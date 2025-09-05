package main

import (
	"os"

	"github.com/tvandinther/gitops-manager/internal/git"
	"github.com/tvandinther/gitops-manager/pkg/flow"
	"github.com/tvandinther/gitops-manager/pkg/gitops"
	"github.com/tvandinther/gitops-manager/pkg/gitops/authenticator"
	"github.com/tvandinther/gitops-manager/pkg/gitops/committer"
	"github.com/tvandinther/gitops-manager/pkg/gitops/copier"
	"github.com/tvandinther/gitops-manager/pkg/gitops/reviewer"
	"github.com/tvandinther/gitops-manager/pkg/server"
)

func main() {
	// repositoryClient, err := gitea.NewClient(&gitea.ClientOptions{
	// 	Host: os.Getenv("GITEA_HOST"),
	// 	AccessTokenAuth: gitea.AccessTokenAuth{
	// 		Username:    os.Getenv("GITEA_USER"),
	// 		AccessToken: os.Getenv("GITEA_ACCESS_TOKEN"),
	// 	},
	// })
	// if err != nil {
	// 	log.Fatalf("failed to create Gitea client: %s", err)
	// }

	giteaAuthenticator := &authenticator.UserPassword{
		Username: os.Getenv("GITEA_USER"),
		Password: os.Getenv("GITEA_ACCESS_TOKEN"),
	}

	gitAuthor := &git.Author{
		Name:  "gitops-manager",
		Email: "gitops-manager@example.com",
	}

	dummyReviewer := &reviewer.Dummy{
		URL:      "https://example.com/review/1",
		Complete: true,
	}

	flow := flow.New(&flow.Strategies{
		RequestAuthorisation: gitops.NoAuthorisation,
		CloneAuthentication:  giteaAuthenticator,
		Branch:               nil,
		FileCopy: &copier.Subpath{
			Path: ".",
		},
		Commit: &committer.Standard{
			Author:        gitAuthor,
			CommitSubject: "Update rendered manifests",
			CommitMessageFn: func(req *gitops.Request) string {
				return "Update rendered manifests"
			},
		},
		CreateReview:   dummyReviewer,
		CompleteReview: dummyReviewer,
	})

	server := server.New(flow, &server.ManagerOpts{
		GitOptions: server.GitOptions{
			Author: gitAuthor,
		},
	}).WithDefaultLogger()

	server.Run()
}
