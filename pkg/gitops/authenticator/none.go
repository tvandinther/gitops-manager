package authenticator

import "net/url"

type None struct{}

func (_ *None) GetAuthenticatedUrl(u *url.URL, sendMsg func(string)) (*url.URL, error) {
	return u, nil
}
