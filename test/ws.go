package test

import (
	"fmt"
	"oauth-static-webserver/http"
	oidc2 "oauth-static-webserver/oidc"
	"os"

	"github.com/oauth2-proxy/mockoidc"
)

func SetupSWS() (func(), *mockoidc.MockOIDC, *http.Webserver, error) {
	m, err := mockoidc.Run()
	if err != nil {
		return nil, nil, nil, err
	}
	rm, contentPath, err := PrepareContentFolder()
	if err != nil {
		_ = m.Shutdown()
		return nil, nil, nil, err
	}
	sessionStorage, err := os.MkdirTemp("", fmt.Sprintf("oauth-static-webserver-session-%s", RandHex(8)))
	if err != nil {
		_ = m.Shutdown()
		rm()
		return nil, nil, nil, err
	}
	cfg := CreateConfig(m, sessionStorage, contentPath)
	oidc, err := oidc2.NewFromConfig(cfg.Content.OIDC.Providers, cfg.Content.OIDC.BaseUrl)
	if err != nil {
		_ = m.Shutdown()
		rm()
		_ = os.RemoveAll(sessionStorage)
		return nil, nil, nil, err
	}
	ws, err := http.NewWebserver(&cfg, oidc)
	if err != nil {
		_ = m.Shutdown()
		rm()
		_ = os.RemoveAll(sessionStorage)
		return nil, nil, nil, err
	}
	return func() {
		_ = m.Shutdown()
		rm()
		_ = os.RemoveAll(sessionStorage)
		_ = ws.Close()
	}, m, ws, nil
}
