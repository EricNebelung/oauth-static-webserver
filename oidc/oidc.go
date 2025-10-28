package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"oauth-static-webserver/config"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"

	log "github.com/sirupsen/logrus"
)

type OIDC struct {
	providers Providers
	baseUrl   string
}

func New(providers Providers, baseUrl string) *OIDC {
	return &OIDC{
		providers: providers,
		baseUrl:   baseUrl,
	}
}

func NewFromConfig(cfg []config.OIDCProvider, baseUrl string) (*OIDC, error) {
	ps, err := newProviders(cfg, baseUrl)
	if err != nil {
		return nil, err
	}
	return New(ps, baseUrl), nil
}

type jwtClaims struct {
	Subject string   `json:"sub"`
	Groups  []string `json:"groups"`
}

// CreateCallbackHandler create a callback handler for all providers by using the
// parameter "provider" to auth on the right OIDC Provider.
func (o *OIDC) CreateCallbackHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := context.Background()

		providerId := c.Param("provider")
		oidcProv, ok := o.providers[providerId]
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
	}
}

// CreateMiddleware create a middleware, which protect all following routes.
// It checks for user auth and redirect to IdP auth url if needed or redirect to an error page.
// The providerId is required to link to the right provider and the user must be in one of the allowedGroups
// to pass the auth test. If the list is empty, then the group check is disabled.
func (o *OIDC) CreateMiddleware(
	providerId string,
	allowedGroups []string,
) (echo.MiddlewareFunc, error) {
	provider, ok := o.providers[providerId]
	if !ok {
		errorMsg := fmt.Errorf("no OIDC provider with ID %s", providerId)
		log.Error(errorMsg)
		return nil, errorMsg
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
				//errUrl := fmt.Sprintf("%s/error/no-permissions.html", o.baseUrl)
				//return c.Redirect(http.StatusFound, errUrl)
				return c.String(http.StatusForbidden, "You do not have the required permissions to access this resource.")
			}

			return next(c)
		}
	}, nil
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
