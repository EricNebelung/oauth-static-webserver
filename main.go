package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	// log level will be set in config processing based on passed env variable
	// for now set to debug for initial startup logs
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func main() {
	log.Info("initializing OAuth-Static-Webserver")

	cfg, err := LoadAndProcessConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(StartServer(cfg))
}

func StartServer(cfg *Config) error {
	oidc, err := NewFromConfig(cfg.Content.OIDC.Providers, cfg.Content.OIDC.BaseUrl)
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
