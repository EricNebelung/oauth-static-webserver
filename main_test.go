package main

import (
	"fmt"
	"oauth-static-webserver/internal/test"
	"os"

	"github.com/oauth2-proxy/mockoidc"
)

func CreateConfig(m *mockoidc.MockOIDC, sessionPath, staticPath string) Config {
	httpPort, err := test.GetFreePort()
	if err != nil {
		panic(err)
	}
	return Config{
		Settings: Settings{
			Host: SettingsHost{Address: "", Port: httpPort},
			Session: SettingsSession{
				Key:            "472347328478392",
				StoreDriver:    "filesystem",
				StoreDirectory: sessionPath,
			},
		},
		Content: ContentConfig{
			OIDC: ContentConfigOIDC{
				BaseUrl: fmt.Sprintf("http://localhost:%d", httpPort),
				Providers: []OIDCProvider{
					{
						Id:           "test-1",
						ConfigUrl:    m.DiscoveryEndpoint(),
						ClientID:     m.ClientID,
						ClientSecret: m.ClientSecret,
					},
				},
			},
			StaticPages: []StaticPage{
				{
					Id:         "page-1",
					Dir:        fmt.Sprintf("%s/page1", staticPath),
					Url:        "/page1",
					Protection: nil,
				},
				{
					Id:  "page-2",
					Dir: fmt.Sprintf("%s/page2", staticPath),
					Url: "/page2",
					Protection: &StaticPageProtection{
						Provider: "test-1",
						Groups:   nil,
					},
				},
				{
					Id:  "page-3",
					Dir: fmt.Sprintf("%s/page3", staticPath),
					Url: "/page3",
					Protection: &StaticPageProtection{
						Provider: "test-1",
						Groups:   []string{"group-test"},
					},
				},
			},
		},
	}
}

func SetupSWS() (*Config, *mockoidc.MockOIDC, *Webserver, error) {
	m, err := mockoidc.Run()
	if err != nil {
		return nil, nil, nil, err
	}
	rm, contentPath, err := test.PrepareContentFolder()
	if err != nil {
		_ = m.Shutdown()
		return nil, nil, nil, err
	}
	sessionStorage, err := os.MkdirTemp("", fmt.Sprintf("oauth-static-webserver-session-%s", test.RandHex(8)))
	if err != nil {
		_ = m.Shutdown()
		rm()
		return nil, nil, nil, err
	}
	cfg := CreateConfig(m, sessionStorage, contentPath)
	oidc, err := NewFromConfig(cfg.Content.OIDC.Providers, cfg.Content.OIDC.BaseUrl)
	if err != nil {
		_ = m.Shutdown()
		rm()
		_ = os.RemoveAll(sessionStorage)
		return nil, nil, nil, err
	}
	ws, err := NewWebserver(&cfg, oidc)
	if err != nil {
		_ = m.Shutdown()
		rm()
		_ = os.RemoveAll(sessionStorage)
		return nil, nil, nil, err
	}
	return &cfg, m, ws, nil
}
