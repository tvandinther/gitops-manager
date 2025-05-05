package gitops

type Mutator interface {
	MutateDir(path string) error
}
