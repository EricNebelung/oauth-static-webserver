package main

import (
	"log"
	"log/slog"
	"oauth-static-webserver/config"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Info("Init: OAuth Static Webserver")

	cfg, err := config.ProcessConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = InitOIDCProviders(cfg.Content.OIDC.Providers)
	if err != nil {
		log.Fatal(err)
	}

	ws, err := NewWebserver(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := ws.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	log.Fatal(ws.Start())
}
