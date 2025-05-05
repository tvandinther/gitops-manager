package git

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func clone(remoteURL *url.URL, directory string, ref plumbing.ReferenceName, depth int) (*git.Repository, error) {
	cloneOpts := &git.CloneOptions{
		URL:           remoteURL.String(),
		Progress:      nil,
		ReferenceName: ref,
		Depth:         depth,
	}

	slog.Debug("cloning repository", "remoteURL", remoteURL.Redacted())

	return git.PlainClone(directory, false, cloneOpts)
}

type author struct {
	Name  string
	Email string
}

func commit(repo *git.Repository, wt *git.Worktree, author author, commitSubject, commitBody string) error {
	var err error

	if wt == nil {
		wt, err = repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}
	}

	commitMsg := fmt.Sprintf("%s\n\n%s", commitSubject, commitBody)

	commit, err := wt.Commit(commitMsg, &git.CommitOptions{
		AllowEmptyCommits: true,
		Author: &object.Signature{
			Name:  author.Name,
			Email: author.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to prepare commit: %w", err)
	}

	obj, err := repo.CommitObject(commit)
	if err != nil {
		return fmt.Errorf("failed to commit object: %w", err)
	}

	slog.Debug("created commit object", "hash", obj.Hash.String(), "authorEmail", obj.Author.Email)

	return nil
}

func push(repo *git.Repository, branchRefName plumbing.ReferenceName) error {
	remoteName := "origin"
	slog.Info("pushing refs", "localRef", branchRefName.Short(), "remoteRef", remoteName)
	err := repo.Push(&git.PushOptions{
		RemoteName: remoteName,
		Progress:   nil,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+%s:%s", branchRefName, branchRefName)),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

func getOrCreateBranch(repo *git.Repository, branchName, sourceBranchName *plumbing.ReferenceName) (*plumbing.Reference, error) {
	var branchRef *plumbing.Reference

	remoteRefName := plumbing.NewRemoteReferenceName("origin", branchName.Short())
	_, err := repo.Reference(remoteRefName, true)

	if err == plumbing.ErrReferenceNotFound {
		if sourceBranchName == nil {
			branchRef, err = createOrphanBranch(repo, *branchName)
			if err != nil {
				return nil, fmt.Errorf("failed to create orphan branch: %w", err)
			}
		} else {
			branchRef, err = createBranch(repo, *branchName, *sourceBranchName)
			if err != nil {
				return nil, fmt.Errorf("failed to create branch: %s: %w", branchName, err)
			}
		}
	} else {
		branchRef, err = createBranch(repo, *branchName, remoteRefName)
		if err != nil {
			return nil, fmt.Errorf("failed to create branch: %s: %w", branchName, err)
		}
	}

	return branchRef, nil
}

func createBranch(repo *git.Repository, branchRefName, headRefName plumbing.ReferenceName) (*plumbing.Reference, error) {
	slog.Debug("creating branch", "branchName", branchRefName)

	headRef, err := repo.Reference(headRefName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get reference %s: %w", headRefName, err)
	}

	ref := plumbing.NewHashReference(branchRefName, headRef.Hash())

	err = repo.Storer.SetReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to save new branch: %w", err)
	}

	return ref, nil
}

func createOrphanBranch(repo *git.Repository, branchRefName plumbing.ReferenceName) (*plumbing.Reference, error) {
	slog.Debug("creating branch", "orphan", true, "branchName", branchRefName)
	symRef := plumbing.NewSymbolicReference(plumbing.HEAD, branchRefName)

	err := repo.Storer.SetReference(symRef)
	if err != nil {
		return nil, fmt.Errorf("failed to save new branch: %w", err)
	}

	return symRef, nil
}
