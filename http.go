package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/boj/redistore"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	log "github.com/sirupsen/logrus"
)

type Webserver struct {
	e    *echo.Echo
	cfg  *Config
	oidc *OIDC

	fsStore    *sessions.FilesystemStore
	redisStore *redistore.RediStore
}

// NewWebserver creates the Echo instanz, the session store and register all middleware and pages.
func NewWebserver(cfg *Config, oidc *OIDC) (*Webserver, error) {
	ws := &Webserver{
		e:    echo.New(),
		cfg:  cfg,
		oidc: oidc,
	}
	err := ws.createSessionStore()
	if err != nil {
		log.WithError(err).Error("Error creating session store")
		return nil, err
	}
	log.Info("Session-Store initialized")

	// register session store
	store, err := ws.getStore()
	if err != nil {
		log.WithError(err).Error("Error getting session store")
		return nil, err
	}
	ws.e.Use(session.Middleware(store))

	// setup webserver routes
	ws.e.GET("/auth/:provider/callback", oidc.CreateCallbackHandler())
	log.Debug("OIDC Auth Callback handler registered")

	// register all pages
	for _, page := range cfg.Content.StaticPages {
		_, err := ws.createStaticPage(ws.e, page)
		if err != nil {
			return nil, err
		}
	}

	// hide some stuff
	ws.e.HideBanner = true
	ws.e.HidePort = true

	return ws, nil
}

// Start the webserver with the Address and Port specified in the config.
func (w *Webserver) Start() error {
	address := w.cfg.Settings.GetWSAddress()
	log.Infof("Listening on %s", address)
	return w.e.Start(address)
}

// StartAsync the webserver in a new goroutine and provide a close function.
func (w *Webserver) StartAsync() error {
	address := w.cfg.Settings.GetWSAddress()
	log.Infof("Listening on %s", address)
	go func() {
		err := w.e.Start(address)
		if err != nil {
			log.WithError(err).Error("Error starting server")
		}
	}()
	return nil
}

func (w *Webserver) Close() error {
	if w.redisStore != nil {
		return w.redisStore.Close()
	}
	return nil
}

// getStore return the existing store or an error, if no store exists.
func (w *Webserver) getStore() (sessions.Store, error) {
	if w.redisStore != nil {
		return w.redisStore, nil
	} else if w.fsStore != nil {
		return w.fsStore, nil
	}
	return nil, errors.New("no session store available")
}

// createSessionStore build the session store from config and set it into the object.
func (w *Webserver) createSessionStore() error {
	cfg := &w.cfg.Settings.Session
	if cfg.StoreDriver == "redis" {
		store, err := redistore.NewRediStore(
			cfg.Redis.PoolSize, "tcp",
			fmt.Sprintf("%s:%d", cfg.Redis.Address, cfg.Redis.Port),
			cfg.Redis.Username, cfg.Redis.Password,
			[]byte(cfg.Key),
		)
		if err != nil || store == nil {
			log.WithError(err).Error("Error creating redis session store")
			return err
		}
		store.Options.MaxAge = 60 * 60 * 24 // 1 day
		w.redisStore = store
		return nil
	} else if cfg.StoreDriver == "filesystem" {
		err := os.MkdirAll(cfg.StoreDirectory, 0700)
		if err != nil {
			log.WithError(err).Error("Error creating Filesystem session store")
			return err
		}

		key := []byte(cfg.Key)
		store := sessions.NewFilesystemStore(cfg.StoreDirectory, key)
		store.Options.MaxAge = 60 * 60 * 24 // 1 day
		w.fsStore = store
		return nil
	}
	log.Errorf("Invalid session store driver: %s", cfg.StoreDriver)
	return errors.New("invalid session store driver")
}

func (w *Webserver) createStaticPage(e *echo.Echo, config StaticPage) (*echo.Group, error) {
	log.WithFields(log.Fields{
		"id":  config.Id,
		"dir": config.Dir,
		"url": config.Url,
	}).Info(
		"Starting registering static page",
	)

	// remove trailing slash if present
	baseContentUrl := strings.TrimRight(config.Url, "/")

	group := e.Group(baseContentUrl)

	// attach protection if configured
	protection := config.Protection
	if protection != nil {
		log.WithFields(log.Fields{
			"id":       config.Id,
			"provider": protection.Provider,
		}).Info("attaching protection for static page")

		protector, err := w.oidc.CreateMiddleware(protection)
		if err != nil {
			log.WithError(err).Error("Error creating protection middleware")
			return nil, err
		}
		group.Use(protector)
	}

	group.Static("/", config.Dir)

	return group, nil
}
