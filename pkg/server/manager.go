package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	igit "github.com/tvandinther/gitops-manager/internal/git"
	"github.com/tvandinther/gitops-manager/internal/util"
	"github.com/tvandinther/gitops-manager/pkg/flow"
	"github.com/tvandinther/gitops-manager/pkg/gitops"
	"github.com/tvandinther/gitops-manager/pkg/progress"
)

type Manager struct {
	flow    *flow.Flow
	report  *progress.Reporter
	options *ManagerOpts
}

type ManagerOpts struct {
	GitOptions GitOptions
}

type GitOptions struct {
	Author     *igit.Author
	CloneDepth int
}

func newManager(reporter *progress.Reporter, flow *flow.Flow, opts *ManagerOpts) *Manager {
	return &Manager{
		flow:    flow,
		report:  reporter,
		options: opts,
	}
}

func (m *Manager) ProcessRequest(ctx context.Context, req *gitops.Request) (*gitops.Response, error) {
	strategies := m.flow.Strategies

	response := &gitops.Response{
		Environment: &gitops.EnvironmentResponse{
			Repository: &req.TargetRepository,
			Name:       req.Environment,
		},
		ReviewResult: &gitops.CreateReviewResult{},
		DryRun:       req.DryRun,
	}

	respondWithError := func(err error) (*gitops.Response, error) {
		slog.Info("responding with error", "error", err)

		response.Error = err.Error()

		return response, err
	}

	m.report.Heading("Verifying request authorisation")
	isAllowed, err := strategies.RequestAuthorisation.Authorise(nil, m.report.BasicProgress)
	if err != nil {
		m.report.Failure("Authorisation failed")
		return respondWithError(fmt.Errorf("failed to perform authorisation: %w", err))
	}
	if !isAllowed {
		m.report.Failure("Request denied")
		return respondWithError(fmt.Errorf("not authorised"))
	}
	m.report.Success("Request authorised")

	gitopsService := gitops.NewService(m.report, gitops.ServiceOptions{
		EnvironmentName:  req.Environment,
		ApplicationName:  req.AppName,
		UpdateIdentifier: req.UpdateIdentifier,
		GitAuthor:        m.options.GitOptions.Author,
	})

	m.report.Heading("Initialising repository")

	environmentBranches := gitopsService.GetEnvironmentBranches()
	response.Environment.RefName = environmentBranches.Trunk.Short()

	slog.Debug("getting authenticated clone URL", "unauthenticatedUrl", req.TargetRepository.URL)
	cloneUrl, err := url.Parse(req.TargetRepository.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target repository URL: %w", err)
	}
	repositoryURL, err := m.flow.Strategies.CloneAuthentication.GetAuthenticatedUrl(cloneUrl, m.report.BasicProgress)
	if err != nil {
		return respondWithError(fmt.Errorf("failed to get authenticated clone URL: %w", err))
	}

	err = gitopsService.InitRepository(repositoryURL, req.Paths.RepositoryDir)
	if err != nil {
		return respondWithError(fmt.Errorf("repository initialization error: %w", err))
	}

	err = gitopsService.PrepareEnvironment()
	if err != nil {
		return respondWithError(fmt.Errorf("environment preparation error: %w", err))
	}

	m.report.Heading("Mutating manifests")

	if len(m.flow.Processors.Mutators) > 0 {
		errs := make([]error, 0)

		nextFns := make([]func(*os.File), len(m.flow.Processors.Mutators)+1)
		nextFns[len(nextFns)-1] = func(file *os.File) {}

		for i, mutator := range m.flow.Processors.Mutators {
			nextFns[i] = func(file *os.File) {
				m.report.Progress("running mutator: %s", mutator.GetTitle())

				inputFile := file
				outputFile := io.NewOffsetWriter(file, 0)

				mutateErr := mutator.MutateFile(
					ctx,
					inputFile,
					outputFile,
					m.report.BasicProgress,
				)
				if mutateErr != nil {
					errs = append(errs, mutateErr)
				}

				nextFns[i+1](file)
			}
		}

		err = filepath.WalkDir(req.Paths.UpdatedManifestsDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if !d.IsDir() {
				m.report.Progress("mutating file: %s", path)

				file, err := os.OpenFile(path, os.O_RDWR, 0644)
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer file.Close()

				nextFns[0](file)
			}

			return nil
		})

		if len(errs) > 0 {
			slog.Error("errors occured during mutation", "count", len(errs))
			m.report.Failure("%d error(s) occured during mutation", len(errs))

			messages := util.Map(errs, func(e error) string {
				if e != nil {
					return e.Error()
				} else {
					return ""
				}
			})
			joinedMessage := strings.Join(messages, "\n")
			err = errors.New(joinedMessage)
			return respondWithError(err)
		}
	} else {
		m.report.Progress("no mutations to be run")
	}

	m.report.Success("Successfully mutated manifests")

	m.report.Heading("Validating manifests")

	if len(m.flow.Processors.Validators) > 0 {
		errs := make([]error, 0)
		successfulValidationResults := make([]*gitops.ValidationResult, 0)
		failedValidationResults := make([]*gitops.ValidationResult, 0)

		err = filepath.WalkDir(req.Paths.UpdatedManifestsDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if !d.IsDir() {
				m.report.Progress("validating file: %s", path)

				file, err := os.OpenFile(path, os.O_RDONLY, 0644)
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer file.Close()

				for _, validator := range m.flow.Processors.Validators {
					slog.Debug("running validator", "title", validator.GetTitle())
					m.report.Progress("running validator %s", validator.GetTitle())

					result, err := validator.ValidateFile(
						ctx,
						file,
						m.report.BasicProgress,
					)
					if result.IsValid {
						successfulValidationResults = append(successfulValidationResults, result)
					} else {
						failedValidationResults = append(failedValidationResults, result)
					}
					errs = append(errs, err)
				}
			}
			return nil
		})

		if len(errs) > 0 {
			slog.Error("errors occured during validation", "count", len(errs))
			m.report.Failure("%d error(s) occured during validation", len(errs))

			respondWithError(errors.New(strings.Join(util.Map(errs, func(e error) string { return e.Error() }), "\n")))
		}

		if len(failedValidationResults) > 0 {
			slog.Info("validations failed", "count", len(failedValidationResults))
			m.report.Failure("%d validation(s) failed", len(failedValidationResults))
			errorStrings := util.Map(failedValidationResults, func(r *gitops.ValidationResult) string {
				return strings.Join(util.Map(r.Errors, func(e error) string { return e.Error() }), "\n")
			})
			respondWithError(fmt.Errorf("invalid manifests: \n%s", strings.Join(errorStrings, "\n\n")))
		}
	} else {
		m.report.Progress("no validations to be run")
	}

	m.report.Success("Successfully validated manifests")

	m.report.Heading("Copying files to the configuration repository")

	err = strategies.FileCopy.CopyFiles(os.DirFS(req.Paths.UpdatedManifestsDir), req.Paths.RepositoryDir, m.report.BasicProgress)
	if err != nil {
		slog.Info("failed to copy files to the configuration repository", "error", err)
		m.report.Failure("Failed to copy files to the configuration repository")

		return respondWithError(fmt.Errorf("failed to copy updated manifests into repository: %w", err))
	}

	m.report.Heading("Committing changes to the configuration repository")

	response.UpdatedFilesCount, err = gitopsService.Commit(req, strategies.Commit)
	if err != nil {
		m.report.Failure("Failed to commit changes to the configuration repository")

		return respondWithError(err)
	}

	if response.UpdatedFilesCount == 0 {
		m.report.Success("No files to be updated in the configuration repository")
		goto End
	} else {
		m.report.Success("Committed %d changes to the configuration repository", response.UpdatedFilesCount)
	}

	m.report.Heading("Pushing changes to the configuration repository")

	if !req.DryRun {
		err = gitopsService.Push(req.TargetRepository)
		if err != nil {
			if err == git.NoErrAlreadyUpToDate {
				m.report.Progress("already up-to-date")
			} else {
				m.report.Failure("Failed to push changes to the configuration repository")

				return respondWithError(err)
			}
		}

		m.report.Success("Pushed %d changes to %s", response.UpdatedFilesCount, req.TargetRepository)
	} else {
		m.report.Progress("performing dry run, changes will not be pushed")
		m.report.Success("(dry-run) Pushed %d changes to %s", response.UpdatedFilesCount, req.TargetRepository)
	}

	m.report.Heading("Creating review")

	if !req.DryRun {
		response.ReviewResult, err = strategies.CreateReview.CreateReview(ctx, req, environmentBranches, m.report.BasicProgress)
		if err != nil {
			slog.Info("failed to create review", "error", err)
			m.report.Failure("Failed to create review")
			return respondWithError(fmt.Errorf("failed to create review: %w", err))
		}

		if !response.ReviewResult.Created {
			m.report.Failure("Failed to create review")
			return respondWithError(fmt.Errorf("failed to create review"))
		}

		m.report.Progress("Review URL: %s", response.ReviewResult.URL)
		m.report.Success("Created review")
	} else {
		response.ReviewResult.Created = true
		m.report.Success("(dry-run) Created review")
	}

	if req.AutoReview {
		m.report.Heading("Completing review")
		if !req.DryRun {
			completed, err := strategies.CompleteReview.CompleteReview(ctx, req, response.ReviewResult, m.report.BasicProgress)
			if err != nil {
				slog.Info("failed to complete review", "error", err)
				m.report.Failure("Failed to complete review")
				return respondWithError(fmt.Errorf("failed to complete review: %w", err))
			}

			response.ReviewResult.Completed = completed

			if !completed {
				m.report.Failure("Failed to complete review")
				return respondWithError(fmt.Errorf("failed to complete review"))
			}

			m.report.Success("Completed review")
		} else {
			response.ReviewResult.Completed = true
			m.report.Success("(dry-run) Completed review")
		}
	}

End:
	response.Msg = "Git operations completed successfully."
	m.report.Success("%s", response.Msg)

	return response, nil
}
