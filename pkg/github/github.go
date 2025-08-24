package github

import "github.com/tvandinther/gitops-manager/internal/manifests"

type Client struct {
	InstallationID int64
	PrivateKeyPath string
}

func (c *Client) Clone(string manifests.Repository) {

}
