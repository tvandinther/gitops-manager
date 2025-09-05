package gitops

type Authorisor interface {
	Authorise(req *Request, sendMsg func(string)) (bool, error)
}
