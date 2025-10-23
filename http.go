package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"oauth-static-webserver/config"
	"os"

	"github.com/boj/redistore"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type Webserver struct {
	e *echo.Echo

	fsStore    *sessions.FilesystemStore
	redisStore *redistore.RediStore
}

// NewWebserver creates the Echo instanz, the session store and register all middleware and pages.
func NewWebserver() (*Webserver, error) {
	ws := &Webserver{
		e: echo.New(),
	}
	err := ws.createSessionStore()
	if err != nil {
		slog.Error("Error creating session store", "err", err)
		return nil, err
	}
	slog.Info("Session-Store initialized")

	// register session store
	store, err := ws.getStore()
	if err != nil {
		slog.Error("Error getting session store", "err", err)
		return nil, err
	}
	ws.e.Use(session.Middleware(store))

	// setup webserver routes
	RegisterCallbackHandler(ws.e)
	slog.Debug("OIDC Auth Callback handler registered")

	// register all pages
	for _, page := range config.Cfg.Content.StaticPages {
		_, err := createStaticPage(ws.e, page)
		if err != nil {
			return nil, err
		}
	}

	// add error pages
	ws.setupErrorPages()

	// hide some stuff
	ws.e.HideBanner = true
	ws.e.HidePort = true

	return ws, nil
}

// Start the webserver with the Address and Port specified in the config.
func (w *Webserver) Start() error {
	address := config.Cfg.Settings.GetWSAddress()
	slog.Info(fmt.Sprintf("Listening on %s", address))
	return w.e.Start(address)
}

func (w *Webserver) Close() error {
	if w.redisStore != nil {
		return w.redisStore.Close()
	}
	return nil
}

//go:embed static
var embeddedFiles embed.FS

func (w *Webserver) setupErrorPages() {
	// get subdirectory to remove "static" in path
	fsError, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatal("Error getting embedded static files", "err", err)
	}
	w.e.StaticFS("/error", fsError)
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
	cfg := config.Cfg.Settings.Session
	if cfg.StoreDriver == "redis" {
		store, err := redistore.NewRediStore(
			cfg.Redis.PoolSize, "tcp",
			fmt.Sprintf("%s:%d", cfg.Redis.Address, cfg.Redis.Port),
			cfg.Redis.Username, cfg.Redis.Password,
			[]byte(cfg.Key),
		)
		if err != nil || store == nil {
			slog.Error("Error creating redis session store", "err", err)
			return err
		}
		store.Options.MaxAge = 60 * 60 * 24 // 1 day
		w.redisStore = store
		return nil
	} else if cfg.StoreDriver != "filesystem" {
		err := os.MkdirAll(cfg.StoreDirectory, 0700)
		if err != nil {
			slog.Error("Error creating Filesystem session store", "err", err)
			return err
		}

		key := []byte(cfg.Key)
		store := sessions.NewFilesystemStore(cfg.StoreDirectory, key)
		store.Options.MaxAge = 60 * 60 * 24 // 1 day
		w.fsStore = store
		return nil
	}
	slog.Error("Invalid session store driver: ", cfg.StoreDriver)
	return errors.New("invalid session store driver")
}

func createStaticPage(e *echo.Echo, config config.StaticPage) (*echo.Group, error) {
	slog.Info(
		"Starting registering static page",
		"id", config.Id,
		"dir", config.Dir,
		"url", config.Url,
	)

	baseUrl := config.Url
	// remove trailing slash if present
	if baseUrl[len(baseUrl)-1] == '/' {
		baseUrl = baseUrl[:len(baseUrl)-1]
	}
	group := e.Group(baseUrl)

	// attach protection if configured
	protection := config.Protection
	if protection != nil {
		slog.Info("attaching protection for static page", "id", config.Id, "provider", protection.Provider)
		group.Use(RequireAuthMiddleware(protection.Provider, protection.Groups))
	}

	group.Static("/", config.Dir)

	return group, nil
}
