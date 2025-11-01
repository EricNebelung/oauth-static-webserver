package main

import (
	"encoding/gob"
)

const sessionName = "oidc_auth_session"
const providerSessionsKey = "oidc_provider_sessions"

type ProviderSession struct {
	ExpiresAt int64          `json:"expires_at"`
	Subject   string         `json:"subject"`
	Groups    []string       `json:"groups"`
	UserInfo  map[string]any `json:"user_info"`
}

func init() {
	// register the custom session type
	gob.Register(ProviderSession{})
	// map format: providerId -> Session for the provider
	gob.Register(map[string]ProviderSession{})
}
