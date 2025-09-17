package targeters

import (
	"fmt"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type BranchOptions struct {
	Prefix        string
	DirectoryName string
	Orphan        bool   // Will take precedence over Upstream if true
	Upstream      string // Must be set if Orphan is false
}

type Branch struct {
	Prefix        string
	DirectoryName string
	Orphan        bool   // Will take precedence over Upstream if true
	Upstream      string // Must be set if Orphan is false
}

func NewBranch(opts BranchOptions) (*Branch, error) {
	if !opts.Orphan && opts.Upstream == "" {
		return nil, fmt.Errorf("targeter misconfigured, upstream branch name cannot be empty when Orphan is false")
	}

	if opts.Orphan {
		opts.Upstream = ""
	}

	b := &Branch{
		Prefix:        opts.Prefix,
		DirectoryName: opts.DirectoryName,
		Orphan:        opts.Orphan,
		Upstream:      opts.Upstream,
	}

	return b, nil
}

func (b *Branch) CreateTarget(req *gitops.Request) (*gitops.Target, error) {
	if !b.Orphan && b.Upstream == "" {
		return nil, fmt.Errorf("targeter misconfigured, upstream branch name cannot be empty when Orphan is false")
	}

	if b.Orphan {
		b.Upstream = ""
	}

	target := &gitops.Target{
		Repository: req.TargetRepository,
		Branch: gitops.TargetBranch{
			Source:         fmt.Sprintf("%s%s-next/%s/%s", b.Prefix, req.Environment, req.AppName, req.UpdateIdentifier),
			Target:         fmt.Sprintf("%s%s", b.Prefix, req.Environment),
			UpstreamSource: b.Upstream,
		},
		Directory: b.DirectoryName,
	}

	return target, nil
}
