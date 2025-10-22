package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type ContentConfig struct {
	OIDC struct {
		CallbackBaseUrl string         `yaml:"callback_base_url" validate:"required,url"`
		Providers       []OIDCProvider `yaml:"providers" validate:"dive,required"`
	} `yaml:"oidc" validate:"required"`
	StaticPages []StaticPage `yaml:"static_pages" validate:"dive,required"`
}

func (c *ContentConfig) Validate(validate *validator.Validate) error {
	err := validateStruct(validate, c)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// check all static page protections reference valid Providers
	for _, staticPage := range c.StaticPages {
		if staticPage.Protection != nil {
			err := validateStruct(validate, staticPage.Protection)
			if err != nil {
				return fmt.Errorf("static page %q protection validation failed: %w", staticPage.Id, err)
			}
		}
	}
	return nil
}

func (c *ContentConfig) Resolve() error {
	for i := range c.OIDC.Providers {
		err := c.OIDC.Providers[i].ResolveConfig()
		if err != nil {
			return err
		}
	}
	return nil
}

type OIDCProvider struct {
	Id           string `yaml:"id" validate:"alphanum"`
	ConfigUrl    string `yaml:"config_url" validate:"required,url"`
	ClientID     string `yaml:"client_id" validate:"alphanum"`
	ClientSecret string `yaml:"client_secret" validate:"alphanum"`
	Callback     string
	IssuerUrl    string
}

func (p *OIDCProvider) ResolveConfig() error {
	var data struct {
		Issuer string `json:"issuer"`
	}
	err := resolveKnownConfig(p.ConfigUrl, &data)
	if err != nil {
		return err
	}
	p.IssuerUrl = data.Issuer

	p.Callback = fmt.Sprintf(
		"%s/auth/%s/callback/",
		RemoveTrailingChar(Cfg.Content.OIDC.CallbackBaseUrl, '/'),
		p.Id,
	)

	return nil
}

func resolveKnownConfig(url string, target any) error {
	response, err := http.Get(url)
	if err != nil {
		slog.Warn("Failed to fetch OIDC provider config", "err", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Warn("Failed to close OIDC provider config response body", "err", err)
		}
	}(response.Body)
	err = json.NewDecoder(response.Body).Decode(target)
	if err != nil {
		slog.Warn("Failed to decode OIDC provider config", "err", err)
		return err
	}
	return nil
}

type StaticPage struct {
	Id         string `yaml:"id" validate:"alphanum"`
	Dir        string `yaml:"dir" validate:"dir"`
	Url        string `yaml:"url" validate:"required,uri"`
	Protection *struct {
		Provider string `yaml:"provider" validate:"alphanum"`
	}
}

func loadContentConfig(path string) (*ContentConfig, error) {
	var contentCfg ContentConfig
	err := loadConfigFromFile(path, &contentCfg)
	if err != nil {
		return nil, err
	}
	return &contentCfg, nil
}

func loadConfigFromFile(path string, contentCfg *ContentConfig) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(file, contentCfg)
}
