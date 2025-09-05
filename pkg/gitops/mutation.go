package gitops

import "context"

type Mutator interface {
	GetTitle() string
	Mutate(ctx context.Context, dir string, setError func(e error), next func(), sendMsg func(string))
}
