package gitops

import (
	"github.com/go-git/go-git/v5"
)

type CommitOptions struct {
	Repository *git.Repository
	Worktree   *git.Worktree
	Request    *Request
	Target     *Target
}

type CommitResponse struct {
	ObjectCount int
}

type Committer interface {
	Commit(opts *CommitOptions, sendMsg func(string)) (*CommitResponse, error)
}
