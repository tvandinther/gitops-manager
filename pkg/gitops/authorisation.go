package gitops

type Authorisor interface {
	Authorise(req *Request, sendMsg func(string)) (bool, error)
}

type StaticAuthorisor struct {
	Allow bool
}

func (a *StaticAuthorisor) Authorise(_ *Request, sendMsg func(string)) (bool, error) {
	if a.Allow {
		sendMsg("always allowing")
	} else {
		sendMsg("always denying")
	}

	return a.Allow, nil
}

var NoAuthorisation = &StaticAuthorisor{Allow: true}
