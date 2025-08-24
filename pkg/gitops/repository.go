package gitops

import "github.com/tvandinther/gitops-manager/internal/manifests"

type Client interface {
	Clone(repository manifests.Repository)
}
