package gitops

import "context"

type Mutator interface {
	GetTitle() string
	Mutate(ctx context.Context, dir string, error func(e error), next func(), sendMsg func(string))
}
