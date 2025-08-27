package authenticator

import "net/url"

type UserPassword struct {
	Username string
	Password string
}

func (a *UserPassword) GetAuthenticatedUrl(u *url.URL, sendMsg func(string)) (*url.URL, error) {
	u.User = url.UserPassword(a.Username, a.Password)
	sendMsg("authenticating with username and password")

	return u, nil
}
