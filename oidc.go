package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"oauth-static-webserver/config"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

const sessionName = "oidc_auth_session"
const providerSessionsKey = "oidc_provider_sessions"

type jwtClaims struct {
	Subject string   `json:"sub"`
	Groups  []string `json:"groups"`
}

type ProviderSession struct {
	ExpiresAt int64    `json:"expires_at"`
	Subject   string   `json:"subject"`
	Groups    []string `json:"groups"`
}

var providers = make(map[string]*oidcProvider)

type oidcProvider struct {
	provider     *oidc.Provider
	oauth2Config oauth2.Config
}

func init() {
	gob.Register(ProviderSession{})
	gob.Register(map[string]ProviderSession{})
}

func newOidcProvider(config config.OIDCProvider) (*oidcProvider, error) {
	ctx := context.Background()

	p, err := oidc.NewProvider(ctx, config.IssuerUrl)
	if err != nil {
		slog.Error("Failed to create OIDC provider ", "err", err)
		return nil, errors.New("failed to create OIDC provider")
	}
	oauth2Config := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.Callback,
		Endpoint:     p.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	return &oidcProvider{
		provider:     p,
		oauth2Config: oauth2Config,
	}, nil
}

func InitOIDCProviders(config []config.OIDCProvider) error {
	slog.Info("Initializing OIDC Providers")
	for _, c := range config {
		slog.Debug("Initializing OIDC Provider", "id", c.Id)
		p, err := newOidcProvider(c)
		if err != nil {
			return err
		}
		providers[c.Id] = p
	}
	return nil
}

// --- Handler und Middleware ---

// RegisterCallbackHandler register the main callback handler for OIDC. It delegates to the appropriate provider handler by the :provider url param.
func RegisterCallbackHandler(e *echo.Echo) {
	e.GET("/auth/:provider/callback", func(c echo.Context) error {
		ctx := context.Background()

		providerId := c.Param("provider")
		oidcProv, ok := providers[providerId]
		if !ok {
			return c.String(http.StatusBadRequest, "Unbekannter OIDC-Provider")
		}

		sess, err := session.Get(sessionName, c)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Fehler beim Abrufen der Session")
		}

		// collect and check state to prevent CSRF
		receivedState := c.QueryParam("state")
		expectedState, ok := sess.Values["state"].(string)
		if !ok || receivedState != expectedState {
			return c.String(http.StatusUnauthorized, "Ungültiger oder fehlender 'state' Parameter")
		}

		code := c.QueryParam("code")
		if code == "" {
			return c.String(http.StatusBadRequest, "Fehlender 'code' Parameter")
		}

		oauth2Token, err := oidcProv.oauth2Config.Exchange(ctx, code)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Fehler beim Token-Austausch: %v", err))
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			return c.String(http.StatusInternalServerError, "Kein ID Token in der Antwort")
		}

		verifier := oidcProv.provider.Verifier(&oidc.Config{ClientID: oidcProv.oauth2Config.ClientID})
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			return c.String(http.StatusUnauthorized, fmt.Sprintf("ID Token Verifizierung fehlgeschlagen: %v", err))
		}

		// collect claims from id token
		var idTokenClaims jwtClaims
		if err := idToken.Claims(&idTokenClaims); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Fehler beim Extrahieren der Claims: %v", err))
		}

		providerSessions, ok := sess.Values[providerSessionsKey].(map[string]ProviderSession)
		if !ok || providerSessions == nil {
			providerSessions = make(map[string]ProviderSession)
		}
		providerSessions[providerId] = ProviderSession{
			ExpiresAt: oauth2Token.Expiry.Unix(),
			Subject:   idTokenClaims.Subject,
			Groups:    idTokenClaims.Groups,
		}
		sess.Values[providerSessionsKey] = providerSessions

		// store original target url
		redirectURL, _ := sess.Values["original_path"].(string)
		if redirectURL == "" {
			redirectURL = "/"
		}
		// save session
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Fehler beim Speichern der Session: %v", err))
		}
		// redirect to original target
		return c.Redirect(http.StatusFound, redirectURL)
	})
}

func RequireAuthMiddleware(providerId string, allowedGroups []string) echo.MiddlewareFunc {
	provider, ok := providers[providerId]
	if !ok {
		panic("Unbekannter OIDC-Provider in RequireAuthMiddleware: " + providerId)
	}
	oauth2Config := provider.oauth2Config

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get(sessionName, c)
			if err != nil {
				return redirectForAuth(oauth2Config, c)
			}

			providerSessions, ok := sess.Values[providerSessionsKey].(map[string]ProviderSession)
			if !ok || providerSessions == nil {
				return redirectForAuth(oauth2Config, c)
			}

			providerSession, ok := providerSessions[providerId]
			pSJ, _ := json.Marshal(providerSession)
			fmt.Println("Provider Session:", string(pSJ))

			if !ok || (providerSession.ExpiresAt > 0 && providerSession.ExpiresAt < time.Now().Unix()) {
				return redirectForAuth(oauth2Config, c)
			}

			if !checkHasOneGroup(allowedGroups, providerSession.Groups) {
				// TODO: resolve error url once and store global
				errUrl := fmt.Sprintf("%s/error/no-permissions.html", config.Cfg.Content.OIDC.BaseUrl)
				return c.Redirect(http.StatusFound, errUrl)
			}

			return next(c)
		}
	}
}

func checkHasOneGroup(allowed, present []string) bool {
	// when no group allowed, then no rule presentGroup rule is present
	if len(allowed) == 0 {
		return true
	}
	for _, presentGroup := range present {
		for _, allowedGroup := range allowed {
			if presentGroup == allowedGroup {
				return true
			}
		}
	}
	return false
}

func redirectForAuth(oauth2Config oauth2.Config, c echo.Context) error {
	fmt.Printf("redirectForAuth called for provider: %s\n", oauth2Config.ClientID)

	sess, err := session.Get(sessionName, c)
	if err != nil {
		log.Printf("Error getting session: %v", err)
		return c.String(http.StatusInternalServerError, "Session cannot be retrieved")
	}

	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		return err
	}
	state := base64.URLEncoding.EncodeToString(b)

	sess.Values["state"] = state
	sess.Values["original_path"] = c.Request().URL.Path

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.String(http.StatusInternalServerError, "cannot save session")
	}

	authURL := oauth2Config.AuthCodeURL(state)
	return c.Redirect(http.StatusFound, authURL)
}
