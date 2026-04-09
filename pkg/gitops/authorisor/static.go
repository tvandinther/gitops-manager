package authorisor

import (
	"log/slog"

	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Static struct {
	Allow bool
}

// NoAuthorisation allows all requests unconditionally. Use only in trusted
// environments where network-level controls already restrict access.
var NoAuthorisation = &Static{Allow: true}

func (a *Static) Authorise(_ *gitops.Request, sendMsg func(string)) (bool, error) {
	if a.Allow {
		slog.Warn("authorisation check bypassed: NoAuthorisation is active — all requests are allowed")
		sendMsg("always allowing")
	} else {
		sendMsg("always denying")
	}

	return a.Allow, nil
}
