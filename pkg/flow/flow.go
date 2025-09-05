package flow

import "github.com/tvandinther/gitops-manager/pkg/gitops"

type Flow struct {
	Strategies *Strategies
	Processors *Processors
}

type Strategies struct {
	RequestAuthorisation gitops.Authorisor
	CloneAuthentication  gitops.URLAuthenticator
	Branch               any // TODO
	FileCopy             gitops.FileCopier
	Commit               gitops.Committer
	// PushAuthentication   gitops.URLAuthenticator
	CreateReview   gitops.Reviewer
	CompleteReview gitops.Reviewer
}

type Processors struct {
	Mutators   []gitops.Mutator
	Validators []gitops.Validator
}

func New(strategies *Strategies) *Flow {
	return &Flow{
		Strategies: strategies,
		Processors: &Processors{
			Mutators:   make([]gitops.Mutator, 0),
			Validators: make([]gitops.Validator, 0),
		},
	}
}

func (f *Flow) AddMutator(m gitops.Mutator) {
	f.Processors.Mutators = append(f.Processors.Mutators, m)
}

// Overwrites any mutators previously added with AddMutator
func (f *Flow) WithMutators(ms ...gitops.Mutator) {
	f.Processors.Mutators = ms
}

func (f *Flow) AddValidator(v gitops.Validator) {
	f.Processors.Validators = append(f.Processors.Validators, v)
}

// Overwrites any validators previously added with AddValidator
func (f *Flow) WithValidators(vs ...gitops.Validator) {
	f.Processors.Validators = vs
}
