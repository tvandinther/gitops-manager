package github

import "github.com/tvandinther/gitops-manager/pkg/gitops"

type Client struct {
	InstallationID int64
	PrivateKeyPath string
}

func (c *Client) GetAuthenticatedCloneUrl(r *gitops.Repository) (string, error) {
	return "", nil
}
