package targeters

import (
	"fmt"
	"path/filepath"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Directory struct {
	Prefix        string // The path prefix to use relative to the parent directory for environment directories. Omit or use "" for no prefix.
	DirectoryName string // The name of the parent directory containing the environment directories. Omit or use "" for none.
	Branch        string // The branch to target the configuration to.
	Orphan        bool   // Whether to create an orphan branch if branch does not exist. Will take precedence over Upstream if true.
	Upstream      string // The upstream to base the branch off if orphan is false. Must be set if Orphan is false.
}

func (d *Directory) CreateTarget(req *gitops.Request) (*gitops.Target, error) {
	directory := filepath.Join(d.DirectoryName, fmt.Sprintf("%s%s", d.Prefix, req.Environment))

	target := &gitops.Target{
		Repository: req.TargetRepository,
		Branch: gitops.TargetBranch{
			Source:         fmt.Sprintf("next/%s/%s/%s", req.Environment, req.AppName, req.UpdateIdentifier),
			Target:         fmt.Sprintf("%s", d.Branch),
			UpstreamSource: d.Upstream,
		},
		Directory: directory,
	}

	return target, nil
}
