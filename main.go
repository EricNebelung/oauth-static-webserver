package main

import (
	"log"
	"log/slog"
	"oauth-static-webserver/config"

	"github.com/go-playground/validator/v10"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Info("Init: OAuth Static Webserver")

	err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err = config.Cfg.Validate(validate)
	if err != nil {
		panic(err)
	}
	slog.Info("Config read and validated successfully")

	err = InitOIDCProviders(config.Cfg.Content.OIDC.Providers)
	if err != nil {
		panic(err)
	}

	log.Fatal(StartWebserver())
}
