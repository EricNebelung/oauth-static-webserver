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
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"

	log "github.com/sirupsen/logrus"
)

type Webserver struct {
	e    *echo.Echo
	cfg  *Config
	oidc *OIDC

	fsStore    *sessions.FilesystemStore
	redisStore *redistore.RediStore

	tmpDir string
}

// NewWebserver creates the Echo instanz, the session store and register all middleware and pages.
func NewWebserver(cfg *Config, oidc *OIDC) (*Webserver, error) {
	ws := &Webserver{
		e:    echo.New(),
		cfg:  cfg,
		oidc: oidc,
	}

	// when TLS and http redirection is enabled, register the redirect handler
	if tls := cfg.Settings.TLS; tls.Enabled && tls.HTTPRedirect {
		ws.e.Pre(middleware.HTTPSRedirect())
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
// It will always start a HTTP2 server, regardless if TLS is configured or not.
// Also, it will use TLS with certs or Auto-TLS if configured in the settings.
func (w *Webserver) Start() error {
	address := w.cfg.Settings.GetWSAddress()
	tls := w.cfg.Settings.TLS

	// cleartext HTTP2 server (H2C)
	if !tls.Enabled {
		s := w.cfg.Settings.HTTP2.GetHttps2Server()
		return w.e.StartH2CServer(address, s)
	}

	// TLS HTTP2 server (H2) with predefined certs
	// when TLS is enabled but AutoTLS is disabled
	if !tls.AutoTLS {
		// check if cert and key files are present
		if tls.CertFile == "" || tls.KeyFile == "" {
			err := errors.New("TLS is enabled but cert or key file is not set")
			log.WithError(err).Error("Error starting server")
			return err
		} else {
			if _, err := os.Stat(tls.CertFile); os.IsNotExist(err) {
				log.WithError(err).Errorf("TLS cert file %s does not exist", tls.CertFile)
				return err
			}
			if _, err := os.Stat(tls.KeyFile); os.IsNotExist(err) {
				log.WithError(err).Errorf("TLS key file %s does not exist", tls.KeyFile)
				return err
			}
		}

		cert, key := tls.CertFile, tls.KeyFile
		return w.e.StartTLS(address, cert, key)
	}

	// TLS HTTP2 server (H2) with AutoTLS
	cacheDir := tls.AutoTLSCertCacheDir
	// when no cache dir is set, create a tmp dir
	if cacheDir == "" {
		tmpDir, err := os.MkdirTemp("", "oauth-static-webserver-autotls")
		if err != nil {
			log.WithError(err).Error("Error creating temp dir for AutoTLS cert cache")
			return err
		}
		w.tmpDir = tmpDir
		cacheDir = tmpDir
		log.Warnf("No AutoTLS cert cache dir set, using temp dir: %s", cacheDir)
	}
	cache := autocert.DirCache(cacheDir)
	w.e.AutoTLSManager.Cache = cache

	log.Infof("Listening on %s", address)
	return w.e.StartAutoTLS(address)
}

// StartAsync the webserver in a new goroutine and provide a close function.
// It calls the Start function in a new goroutine and logs any error returned.
func (w *Webserver) StartAsync() error {
	go func() {
		err := w.Start()
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
	if len(w.tmpDir) > 0 {
		err := os.RemoveAll(w.tmpDir)
		if err != nil {
			log.WithError(err).Error("Error removing temp dir")
			return err
		}
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
