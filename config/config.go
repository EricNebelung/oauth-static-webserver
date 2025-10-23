package config

import (
	"log"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Settings Settings
	Content  ContentConfig
}

func loadConfig() (*Config, error) {
	cfg := new(Config)
	// first load settings from env
	settings, err := loadSettingsFromEnv()
	if err != nil {
		help, errHelp := cleanenv.GetDescription(&cfg, nil)
		if errHelp != nil {
			slog.Error("can not get help text", errHelp.Error)
		} else {
			slog.Info("Configuration help:\n" + help)
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

func ProcessConfig() (*Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err = cfg.Validate(validate)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("Config read and validated successfully")

	err = cfg.Resolve()
	if err != nil {
		slog.Error("Error resolving config", "err", err)
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate(validate *validator.Validate) error {
	return c.Content.Validate(validate)
}

func (c *Config) Resolve() error {
	return c.Content.Resolve()
}
