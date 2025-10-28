package test

import "github.com/oauth2-proxy/mockoidc"

var (
	User1 = &mockoidc.MockUser{
		Subject:           "1",
		PreferredUsername: "mocker1",
		Groups:            []string{"group-test"},
	}
	User2 = &mockoidc.MockUser{
		Subject:           "2",
		PreferredUsername: "mocker2",
		Groups:            nil,
	}
)
