package targeters

import (
	"fmt"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Repository struct {
	MapRepositoryFn func(environment string) (gitops.Repository, error) // A function to map an environment string to a repository.
	DirectoryName   string                                              // The name of the parent directory containing the environment directories. Omit or use "" for none.
	Branch          string                                              // The branch to target the configuration to.
	Orphan          bool                                                // Whether to create an orphan branch if branch does not exist. Will take precedence over Upstream if true.
	Upstream        string                                              // The upstream to base the branch off if orphan is false. Must be set if Orphan is false.
}

func (r *Repository) CreateTarget(req *gitops.Request) (*gitops.Target, error) {
	repository, err := r.MapRepositoryFn(req.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to map environment to a repository: %w", err)
	}

	target := &gitops.Target{
		Repository: repository,
		Branch: gitops.TargetBranch{
			Source:         fmt.Sprintf("next/%s/%s/%s", req.Environment, req.AppName, req.UpdateIdentifier),
			Target:         fmt.Sprintf("%s", r.Branch),
			UpstreamSource: r.Upstream,
		},
		Directory: r.DirectoryName,
	}

	return target, nil
}
