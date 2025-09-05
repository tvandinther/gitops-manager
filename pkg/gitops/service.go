package gitops

import (
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"

	igit "github.com/tvandinther/gitops-manager/internal/git"
	"github.com/tvandinther/gitops-manager/pkg/progress"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type ServiceOptions struct {
	EnvironmentName  string
	ApplicationName  string
	UpdateIdentifier string
	GitAuthor        *igit.Author
}

type Service struct {
	report              *progress.Reporter
	remoteURL           *url.URL
	repositoryDirectory string
	repository          *git.Repository
	worktree            *git.Worktree
	environmentConfig   environmentConfig
	cloneDepth          int
	author              *igit.Author
}

type environmentConfig struct {
	branches         EnvironmentBranches
	environmentName  string
	applicationName  string
	updateIdentifier string
}

type EnvironmentBranches struct {
	Trunk plumbing.ReferenceName
	Next  plumbing.ReferenceName
}

func NewService(reporter *progress.Reporter, opts ServiceOptions) *Service {
	environmentBranches := getEnvironmentBranchRefNames(opts.EnvironmentName, opts.ApplicationName, opts.UpdateIdentifier)

	return &Service{
		environmentConfig: environmentConfig{
			environmentName:  opts.EnvironmentName,
			applicationName:  opts.ApplicationName,
			updateIdentifier: opts.UpdateIdentifier,
			branches:         environmentBranches,
		},
		report:     reporter,
		cloneDepth: 1,
		author:     opts.GitAuthor,
	}
}

func (s *Service) InitRepository(remoteURL *url.URL, directory string) error {
	s.remoteURL = remoteURL
	s.repositoryDirectory = directory

	var err error

	defer func() {
		r := remoteURL.Path

		s.report.Result(err, progress.Result{
			Success: fmt.Sprintf("Successfully initialised %s", r),
			Failure: fmt.Sprintf("Failed to initialise %s", r),
		})
	}()

	clone := func(ref plumbing.ReferenceName) (*git.Repository, error) {
		cloneOpts := &git.CloneOptions{
			URL:           s.remoteURL.String(),
			Progress:      nil,
			ReferenceName: ref,
			Depth:         s.cloneDepth,
		}

		s.report.Progress("cloning %s#%s", remoteURL.Redacted(), ref)

		return git.PlainClone(s.repositoryDirectory, false, cloneOpts)
	}

	repo, err := clone(s.environmentConfig.branches.Trunk)
	if err == plumbing.ErrReferenceNotFound {
		s.report.Progress("%s not found", s.environmentConfig.branches.Trunk)
		s.report.Progress("cloning the default ref to initialise %s", s.environmentConfig.branches.Trunk)
		repo, err = clone(plumbing.Main)
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		s.report.Progress("initialising environment branch")
		environmentBranch, err := initialiseEnvironmentBranch(repo, s.environmentConfig.branches, s.author)
		if err != nil {
			return fmt.Errorf("failed to initialise environment branch: %w", err)
		}

		err = igit.Push(repo, environmentBranch.Target())
		if err != nil {
			return err
		}
	}
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	cfg := config.NewConfig()
	cfg.User = struct {
		Name  string
		Email string
	}{
		Name:  s.author.Name,
		Email: s.author.Email,
	}

	err = repo.SetConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL.String()},
	})

	s.repository = repo

	return nil
}

func initialiseEnvironmentBranch(repo *git.Repository, environmentBranches EnvironmentBranches, author *igit.Author) (*plumbing.Reference, error) {
	ref, err := igit.CreateOrphanBranch(repo, environmentBranches.Trunk)
	if err != nil {
		return nil, fmt.Errorf("failed to create orphan branch: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}
	err = wt.RemoveGlob("*")
	if err != nil && err != git.ErrGlobNoMatches {
		return nil, fmt.Errorf("failed to clear worktree: %w", err)
	}

	err = igit.Commit(repo, wt, author, "Initialise empty environment", fmt.Sprintf("Initialising %s", environmentBranches.Trunk.Short()))
	if err != nil {
		return nil, fmt.Errorf("failed to make initial environment commit: %w", err)
	}

	return ref, nil
}

func (s *Service) GetEnvironmentBranches() *EnvironmentBranches {
	return &s.environmentConfig.branches
}

func getEnvironmentBranchRefNames(environment, appName, updateIdentifier string) EnvironmentBranches {
	prefix := "environment/"
	trunkBranch := prefix + environment
	nextBranch := fmt.Sprintf("%s%s-next/%s/%s", prefix, environment, appName, updateIdentifier)

	return EnvironmentBranches{
		Trunk: plumbing.NewBranchReferenceName(trunkBranch),
		Next:  plumbing.NewBranchReferenceName(nextBranch),
	}
}

func (s *Service) PrepareEnvironment() error {
	environmentBranches := s.environmentConfig.branches

	err := s.repository.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Progress:   nil,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:%s", environmentBranches.Trunk, environmentBranches.Trunk)),
			config.RefSpec(fmt.Sprintf("%s:%s", environmentBranches.Next, environmentBranches.Next)),
		},
		Depth: s.cloneDepth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate && !strings.Contains(err.Error(), "couldn't find remote ref") {
		return fmt.Errorf("failed to fetch origin: %w", err)
	}

	nextBranchRef, err := igit.GetOrCreateBranch(s.repository, &environmentBranches.Next, &environmentBranches.Trunk)
	if err != nil {
		return fmt.Errorf("failed to get or create environment branch: %w", err)
	}

	wt, err := s.repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	wt.Checkout(&git.CheckoutOptions{
		Branch: nextBranchRef.Name(),
	})

	pattern := filepath.Join("manifests", s.environmentConfig.applicationName, "*")
	err = wt.RemoveGlob(pattern)
	if err != nil && err != git.ErrGlobNoMatches {
		return fmt.Errorf("failed to remove files from %s: %w", pattern, err)
	}

	s.worktree = wt

	return nil
}

func (s *Service) Commit(req *Request, committer Committer) (int, error) {
	commitOptions := &CommitOptions{
		// WorkingDirectory: s.repositoryDirectory,
		Repository: s.repository,
		Worktree:   s.worktree,
		Request:    req,
	}
	slog.Debug("committing changes to the configuration repository", "options", commitOptions)
	resp, err := committer.Commit(commitOptions, s.report.BasicProgress)
	if err != nil {
		return 0, fmt.Errorf("failed to commit: %w", err)
	}

	return resp.ObjectCount, nil
}

func (s *Service) Push(configRepository Repository) error {
	environmentBranches := s.environmentConfig.branches
	s.report.Progress("pushing %s to %s", environmentBranches.Next, configRepository)
	err := igit.Push(s.repository, environmentBranches.Next)
	if err != nil {
		return err
	}

	return nil
}
