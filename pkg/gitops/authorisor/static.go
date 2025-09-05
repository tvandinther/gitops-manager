package authorisor

import "github.com/tvandinther/gitops-manager/pkg/gitops"

type Static struct {
	Allow bool
}

var NoAuthorisation = &Static{Allow: true}

func (a *Static) Authorise(_ *gitops.Request, sendMsg func(string)) (bool, error) {
	if a.Allow {
		sendMsg("always allowing")
	} else {
		sendMsg("always denying")
	}

	return a.Allow, nil
}
