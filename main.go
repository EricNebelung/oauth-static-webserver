package main

import (
	"oauth-static-webserver/config"
	oidc2 "oauth-static-webserver/oidc"
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func main() {
	log.Info("initializing OAuth-Static-Webserver")

	cfg, err := config.ProcessConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(StartServer(cfg))
}

func StartServer(cfg *config.Config) error {
	//err := InitOIDCProviders(cfg.Content.OIDC.Providers)
	//if err != nil {
	//	return err
	//}
	oidc, err := oidc2.NewFromConfig(cfg.Content.OIDC.Providers, cfg.Content.OIDC.BaseUrl)
	if err != nil {
		return err
	}

	ws, err := NewWebserver(cfg, oidc)
	if err != nil {
		return err
	}
	defer func() {
		err := ws.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	return ws.Start()
}
