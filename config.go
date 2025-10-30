package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"
)

// --- Config type declaration ---

// Config is the main configuration struct containing settings and content config
type Config struct {
	// contains the main settings for the webserver
	Settings Settings
	// contains the configuration for the pages and security
	Content ContentConfig
}

type Settings struct {
	Host       SettingsHost    `env-prefix:"HOST_"`
	Session    SettingsSession `env-prefix:"SESSION_"`
	ConfigPath string          `env:"CONFIG_PATH" env-default:"/etc/oauth-resource-proxy/config.yaml"`
}

type SettingsSession struct {
	Key string `env:"KEY"`
	// Possible values: "filesystem", "redis"
	StoreDriver string `env:"STORE_DRIVER" env-default:"filesystem" env-description:"Session store driver: filesystem or redis"`
	// when redis
	Redis struct {
		Address  string `env:"ADDRESS"`
		Port     int    `env:"PORT" env-default:"6379"`
		Username string `env:"USERNAME"`
		Password string `env:"PASSWORD"`
		DB       int    `env:"DB" env-default:"0"`
		PoolSize int    `env:"POOL_SIZE" env-default:"10"`
	} `env-prefix:"REDIS_"`
	// when filesystem
	StoreDirectory string `env:"STORE_DIRECTORY"`
}

type SettingsHost struct {
	Address string `env:"ADDRESS"`
	Port    int    `env:"PORT" env-default:"8080"`
}

type ContentConfig struct {
	OIDC        ContentConfigOIDC `yaml:"oidc" validate:"required"`
	StaticPages []StaticPage      `yaml:"static_pages" validate:"dive,required"`
}

type ContentConfigOIDC struct {
	BaseUrl   string         `yaml:"base_url" validate:"required,url"`
	Providers []OIDCProvider `yaml:"providers" validate:"dive,required"`
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

// --- Config loading and processing ---

// loadConfig loads the configuration from environment variables and config file
func loadConfig() (*Config, error) {
	cfg := new(Config)
	// first load settings from env
	settings, err := loadSettingsFromEnv()
	if err != nil {
		help, errHelp := cleanenv.GetDescription(&cfg, nil)
		if errHelp != nil {
			log.WithError(err).WithError(errHelp).Error("can not get help text")
		} else {
			log.WithError(err).Fatal(help)
		}
		return nil, err
	}
	cfg.Settings = settings
	// then load content config from file
	contentCfg, err := loadContentConfig(settings.ConfigPath)
	if err != nil {
		return nil, err
	}
	cfg.Content = *contentCfg
	return cfg, nil
}

// LoadAndProcessConfig loads, validates and resolves the configuration
func LoadAndProcessConfig() (*Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		log.WithError(err).Fatal("error loading config")
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err = cfg.Validate(validate)
	if err != nil {
		log.WithError(err).Fatal("configuration are not valid")
	}
	log.Info("Config read and validated successfully")

	err = cfg.Process()
	if err != nil {
		log.WithError(err).Error("Error resolving config")
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate(validate *validator.Validate) error {
	return c.Content.Validate(validate)
}

func (c *Config) Process() error {
	return c.Content.Process()
}

func loadSettingsFromEnv() (Settings, error) {
	var settings Settings
	err := cleanenv.ReadEnv(&settings)
	return settings, err
}

func (s Settings) GetWSAddress() string {
	return fmt.Sprintf("%s:%d", s.Host.Address, s.Host.Port)
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

func (c *ContentConfig) Process() error {
	c.OIDC.BaseUrl = strings.TrimRight(c.OIDC.BaseUrl, "/")
	return nil
}

// --- helper and utility functions ---

func validateStruct(validate *validator.Validate, value any) error {
	err := validate.Struct(value)
	if err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			fmt.Println(err)
			return err
		}
	}
	return err
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
