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

func LoadConfig() error {
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

func (c Config) Validate(validate *validator.Validate) error {
	return c.Content.Validate(validate)
}
