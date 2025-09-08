package main

import (
	"fmt"
	"log"
	"os"

	"code.gitea.io/sdk/gitea"
	"github.com/tvandinther/gitops-manager/internal/git"
	"github.com/tvandinther/gitops-manager/pkg/flow"
	"github.com/tvandinther/gitops-manager/pkg/gitops"
	"github.com/tvandinther/gitops-manager/pkg/gitops/authenticator"
	"github.com/tvandinther/gitops-manager/pkg/gitops/authorisor"
	"github.com/tvandinther/gitops-manager/pkg/gitops/committer"
	"github.com/tvandinther/gitops-manager/pkg/gitops/copier"
	"github.com/tvandinther/gitops-manager/pkg/gitops/mutators"
	"github.com/tvandinther/gitops-manager/pkg/gitops/reviewer"
	"github.com/tvandinther/gitops-manager/pkg/gitops/targeters"
	"github.com/tvandinther/gitops-manager/pkg/gitops/validators"
	"github.com/tvandinther/gitops-manager/pkg/server"
)

func main() {
	giteaClient, err := gitea.NewClient(os.Getenv("GITEA_HOST"), gitea.SetBasicAuth(os.Getenv("GITEA_USER"), os.Getenv("GITEA_ACCESS_TOKEN")))
	if err != nil {
		log.Fatalf("failed to create Gitea client: %s", err)
	}

	authenticator := &authenticator.UserPassword{
		Username: os.Getenv("GITEA_USER"),
		Password: os.Getenv("GITEA_ACCESS_TOKEN"),
	}

	gitAuthor := &git.Author{
		Name:  "gitops-manager",
		Email: "gitops-manager@example.com",
	}

	reviewer := &reviewer.Gitea{
		Client: giteaClient,
		MergeOptions: &gitea.MergePullRequestOption{
			Style:                  gitea.MergeStyleRebase,
			DeleteBranchAfterMerge: true,
		},
	}

	flow := flow.New(&flow.Strategies{
		RequestAuthorisation: authorisor.NoAuthorisation,
		CloneAuthentication:  authenticator,
		Target:               &targeters.Branch{Prefix: "environment", DirectoryName: "manifests", Orphan: true},
		FileCopy: &copier.Subpath{
			Path: ".",
		},
		Commit: &committer.Standard{
			Author:        gitAuthor,
			CommitSubject: "Update rendered manifests",
			CommitMessageFn: func(req *gitops.Request) string {
				return fmt.Sprintf("Update rendered manifests for %s", req.AppName)
			},
		},
		CreateReview:   reviewer,
		CompleteReview: reviewer,
	})

	flow.WithMutators(&mutators.HelmHooksToArgoCD{})
	flow.WithValidators(&validators.EmptyFile{})

	server := server.New(flow, &server.ManagerOpts{
		GitOptions: server.GitOptions{
			Author: gitAuthor,
		},
	}).WithDefaultLogger()

	server.Run()
}
