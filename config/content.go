package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type ContentConfig struct {
	OIDC        ContentConfigOIDC `yaml:"oidc" validate:"required"`
	StaticPages []StaticPage      `yaml:"static_pages" validate:"dive,required"`
}

type ContentConfigOIDC struct {
	BaseUrl   string         `yaml:"base_url" validate:"required,url"`
	Providers []OIDCProvider `yaml:"providers" validate:"dive,required"`
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
	c.OIDC.BaseUrl = strings.TrimRight(c.OIDC.BaseUrl, "/")
	return nil
}

type OIDCProvider struct {
	Id           string `yaml:"id" validate:"alphanum"`
	ConfigUrl    string `yaml:"config_url" validate:"required,url"`
	ClientID     string `yaml:"client_id" validate:"alphanum"`
	ClientSecret string `yaml:"client_secret" validate:"alphanum"`
}

type StaticPage struct {
	Id         string                `yaml:"id" validate:"alphanum"`
	Dir        string                `yaml:"dir" validate:"dir"`
	Url        string                `yaml:"url" validate:"required,uri"`
	Protection *StaticPageProtection `yaml:"protection"`
}

type StaticPageProtection struct {
	Provider string   `yaml:"provider" validate:"alphanum"`
	Groups   []string `yaml:"groups" validate:"dive,alphanum"`
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
