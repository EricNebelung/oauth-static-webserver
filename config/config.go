package config

import (
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

var Cfg Config

type Config struct {
	Settings Settings
	Content  ContentConfig
}

func loadConfig() error {
	// first load settings from env
	settings, err := loadSettingsFromEnv()
	if err != nil {
		help, errHelp := cleanenv.GetDescription(&Cfg, nil)
		if errHelp != nil {
			slog.Error("can not get help text", errHelp.Error)
		} else {
			slog.Info("Configuration help:\n" + help)
		}
		return err
	}
	Cfg.Settings = settings
	// then load content config from file
	contentCfg, err := loadContentConfig(settings.ConfigPath)
	if err != nil {
		return err
	}
	Cfg.Content = *contentCfg
	return nil
}

func ProcessConfig() error {
	err := loadConfig()
	if err != nil {
		panic(err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err = Cfg.Validate(validate)
	if err != nil {
		panic(err)
	}
	slog.Info("Config read and validated successfully")

	err = Cfg.Resolve()
	if err != nil {
		slog.Error("Error resolving config", "err", err)
		return err
	}
	return nil
}

func (c *Config) Validate(validate *validator.Validate) error {
	return c.Content.Validate(validate)
}

func (c *Config) Resolve() error {
	return c.Content.Resolve()
}
