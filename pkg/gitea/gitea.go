package gitea

import (
	"fmt"
	"net/url"

	"code.gitea.io/sdk/gitea"
	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

type Client struct {
	Client          *gitea.Client
	Host            string
	accessTokenAuth AccessTokenAuth
}

type AccessTokenAuth struct {
	Username    string
	AccessToken string
}

type ClientOptions struct {
	Host            string
	AccessTokenAuth AccessTokenAuth
}

func NewClient(opts *ClientOptions) (*Client, error) {
	giteaClient, _ := gitea.NewClient(opts.Host, gitea.SetBasicAuth(opts.AccessTokenAuth.Username, opts.AccessTokenAuth.AccessToken))
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create Gitea client: %w", err)
	// }

	return &Client{
		Client:          giteaClient,
		Host:            opts.Host,
		accessTokenAuth: opts.AccessTokenAuth,
	}, nil
}

func (c *Client) GetAuthenticatedCloneUrl(repository *gitops.Repository) (*url.URL, error) {
	repositoryURL, err := url.Parse(repository.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	repositoryURL.User = url.UserPassword(c.accessTokenAuth.Username, c.accessTokenAuth.AccessToken)

	return repositoryURL, nil
}
