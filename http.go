package main

import (
	"errors"
	"fmt"
	"log/slog"
	"oauth-static-webserver/config"
	"os"

	"github.com/boj/redistore"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

func StartWebserver() error {
	e := echo.New()
	store, err := createSessionStore()
	if err != nil {
		slog.Error("Error creating session store", "err", err)
		panic(err)
	}
	defer func(store *sessionStore) {
		err := store.Close()
		if err != nil {
			slog.Error("Error closing session store", "err", err)
		}
	}(store)
	slog.Info("Session-Store initialized")

	sessStore, err := store.GetStore()
	if err != nil {
		slog.Error("Error getting session store", "err", err)
		panic(err)
	}
	e.Use(session.Middleware(sessStore))
	slog.Debug("Session-Store middleware attached")

	RegisterCallbackHandler(e)
	slog.Debug("OIDC Auth Callback handler registered")

	for _, page := range config.Cfg.Content.StaticPages {
		_, err := createStaticPage(e, page)
		if err != nil {
			panic(err)
		}
	}

	e.HideBanner = true
	e.HidePort = true
	slog.Info("Starting webserver", "address", config.Cfg.Settings.GetWSAddress())
	return e.Start(config.Cfg.Settings.GetWSAddress())
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
		group.Use(RequireAuthMiddleware(protection.Provider))
	}

	group.Static("/", config.Dir)

	return group, nil
}

type sessionStore struct {
	FsStore    *sessions.FilesystemStore
	RedisStore *redistore.RediStore
}

func (s sessionStore) Close() error {
	if s.RedisStore != nil {
		return s.RedisStore.Close()
	}
	return nil
}

func (s sessionStore) GetStore() (sessions.Store, error) {
	if s.RedisStore != nil {
		return s.RedisStore, nil
	} else if s.FsStore != nil {
		return s.FsStore, nil
	}
	return nil, errors.New("no session store available")
}

func createSessionStore() (*sessionStore, error) {
	// TODO: add support for redis
	cfg := config.Cfg.Settings.Session
	if cfg.StoreDriver == "redis" {
		store, err := redistore.NewRediStore(
			cfg.Redis.PoolSize, "tcp",
			fmt.Sprintf("%s:%d", cfg.Redis.Address, cfg.Redis.Port),
			cfg.Redis.Username, cfg.Redis.Password,
			[]byte(cfg.Key),
		)
		store.Options.MaxAge = 60 * 60 * 24 // 1 day
		if err != nil {
			slog.Error("Error creating redis session store", "err", err)
			return nil, err
		}
		return &sessionStore{RedisStore: store}, nil
	} else if cfg.StoreDriver != "filesystem" {
		err := os.MkdirAll(cfg.StoreDirectory, 0700)
		if err != nil {
			slog.Error("Error creating Filesystem session store", "err", err)
			return nil, err
		}

		key := []byte(cfg.Key)
		store := sessions.NewFilesystemStore(cfg.StoreDirectory, key)
		store.Options.MaxAge = 60 * 60 * 24 // 1 day
		return &sessionStore{FsStore: store}, nil
	}
	slog.Error("Invalid session store driver: ", cfg.StoreDriver)
	return nil, errors.New("invalid session store driver")
}
