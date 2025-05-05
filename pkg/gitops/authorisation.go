package gitops

type Authorisor interface {
	Authorise(any) (bool, error)
}
