package test

import (
	"fmt"
	"oauth-static-webserver/config"

	"github.com/oauth2-proxy/mockoidc"
)

func CreateConfig(m *mockoidc.MockOIDC, sessionPath, staticPath string) config.Config {
	return config.Config{
		Settings: config.Settings{
			Host: config.SettingsHost{Address: "", Port: 8454},
			Session: config.SettingsSession{
				Key:            "472347328478392",
				StoreDriver:    "filesystem",
				StoreDirectory: sessionPath,
			},
		},
		Content: config.ContentConfig{
			OIDC: config.ContentConfigOIDC{
				BaseUrl: "http://localhost:8454",
				Providers: []config.OIDCProvider{
					{
						Id:           "test-1",
						ConfigUrl:    m.DiscoveryEndpoint(),
						ClientID:     m.ClientID,
						ClientSecret: m.ClientSecret,
					},
				},
			},
			StaticPages: []config.StaticPage{
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
					Protection: &config.StaticPageProtection{
						Provider: "test-1",
						Groups:   nil,
					},
				},
				{
					Id:  "page-3",
					Dir: fmt.Sprintf("%s/page3", staticPath),
					Url: "/page3",
					Protection: &config.StaticPageProtection{
						Provider: "test-1",
						Groups:   []string{"group-test"},
					},
				},
			},
		},
	}
}
