package committer

import (
	"fmt"
	"log/slog"
	"path/filepath"

	gitUtil "github.com/tvandinther/gitops-manager/internal/git"
	"github.com/tvandinther/gitops-manager/pkg/gitops"
	pgit "github.com/tvandinther/gitops-manager/pkg/gitops/git"
)

type Standard struct {
	Author          *pgit.Author
	CommitSubject   string
	CommitMessageFn func(req *gitops.Request) string
}

func (c *Standard) Commit(opts *gitops.CommitOptions, sendMsg func(string)) (*gitops.CommitResponse, error) {
	pattern := filepath.Join(opts.Target.Directory, "*")
	slog.Debug("adding files to worktree", "pattern", pattern)
	sendMsg("adding updated manifests to the current git worktree")
	err := opts.Worktree.AddGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to add files: %w", err)
	}

	status, err := opts.Worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}
	// slog.Debug("git status", "isClean", status.IsClean(), "string", status.String())

	objectCount := len(status)
	sendMsg(fmt.Sprintf("found %d changed objects", objectCount))

	repoHead, err := opts.Repository.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository HEAD: %w", err)
	}

	if !status.IsClean() {
		sendMsg(fmt.Sprintf("comitting %d objects", objectCount))
		commitBody := c.CommitMessageFn(opts.Request)

		err := gitUtil.Commit(opts.Repository, nil, c.Author, c.CommitSubject, commitBody)
		sendMsg(fmt.Sprintf("committed %d objects to %s", objectCount, repoHead.Name().Short()))
		if err != nil {
			return nil, fmt.Errorf("failed to update manifests: %w", err)
		}
	} else {
		sendMsg("no changes to commit")
	}

	// slog.Info("committed objects", "objectCount", objectCount, "refName", repoHead.Name())

	return &gitops.CommitResponse{
		ObjectCount: objectCount,
	}, nil
}
