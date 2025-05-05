package gitops

type Authenticator interface {
	Authenticate(any) (bool, error)
}
