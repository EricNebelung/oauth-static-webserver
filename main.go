package main

import (
	"log"
	"log/slog"
	"oauth-static-webserver/config"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Info("Init: OAuth Static Webserver")

	err := config.ProcessConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = InitOIDCProviders(config.Cfg.Content.OIDC.Providers)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(StartWebserver())
}
