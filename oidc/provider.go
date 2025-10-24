package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"oauth-static-webserver/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	log "github.com/sirupsen/logrus"
)

type Providers map[string]*Provider

// newProviders creates all configured providers.
func newProviders(cfg []config.OIDCProvider, baseUrl string) (Providers, error) {
	p := make(Providers)
	for _, c := range cfg {
		provider, err := newProvider(c, baseUrl)
		if err != nil {
			return nil, err
		}
		p[c.Id] = provider
	}
	return p, nil
}

type Provider struct {
	provider     *oidc.Provider
	oauth2Config oauth2.Config
	cfg          ProviderConfig
}

type ProviderConfig struct {
	config.OIDCProvider
	IssuerUrl string `json:"issuer"`
}

// newProvider creates the internal oidc provider.
// It will fetch all important data from the well-known url.
// Then the oidc.Provider and the required oauth2.Config will be created.
// The callback-url requires the base url as base to build.
func newProvider(cfg config.OIDCProvider, baseUrl string) (*Provider, error) {
	p := new(Provider)
	p.cfg = ProviderConfig{OIDCProvider: cfg}
	err := p.cfg.resolveFromIdP()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	p.provider, err = oidc.NewProvider(ctx, p.cfg.IssuerUrl)
	if err != nil {
		log.WithField("providerId", p.cfg.Id).
			WithError(err).
			Error("Failed to create oidc provider.")
		return nil, err
	}
	p.oauth2Config = oauth2.Config{
		ClientID:     p.cfg.ClientID,
		ClientSecret: p.cfg.ClientSecret,
		RedirectURL:  fmt.Sprintf("%s/auth/%s/callback", baseUrl, p.cfg.Id),
		Endpoint:     p.provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	return p, nil
}

// resolveFromIdP fetch the data from the well-known url and decode it into itself.
func (c *ProviderConfig) resolveFromIdP() error {
	return fetchAndDecodeJson(c.ConfigUrl, c)
}

// fetchAndDecodeJson get the content at the url and decode the json into the target.
// It will log the error and return it, when an error occurs.
func fetchAndDecodeJson(url string, target any) error {
	if target == nil {
		err := errors.New("target cannot be nil")
		log.WithField("url", url).
			WithError(err).
			Error("Error while fetching oidc provider config. The target does not exists!")
		return err
	}

	response, err := http.Get(url)
	if err != nil {
		log.WithField("url", url).WithError(err).Warn("Failed to fetch data from url!")
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithField("url", url).WithError(err).Warn("Failed to close body!")
		}
	}(response.Body)
	err = json.NewDecoder(response.Body).Decode(target)
	if err != nil {
		log.WithField("url", url).WithError(err).Warn("Failed to decode json!")
		return err
	}
	return nil
}
