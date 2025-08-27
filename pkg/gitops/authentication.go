package gitops

import "net/url"

type URLAuthenticator interface {
	GetAuthenticatedUrl(url *url.URL, sendMsg func(string)) (*url.URL, error)
}
