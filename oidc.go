package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
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

func NewFromConfig(cfg []OIDCProvider, baseUrl string) (*OIDC, error) {
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
			log.Debugf("OIDC provider %s not found", providerId)
			return c.String(http.StatusBadRequest, "unknown OIDC provider")
		}

		sess, err := session.Get(sessionName, c)
		if err != nil {
			log.WithError(err).Error("Failed to get session")
			return c.String(http.StatusInternalServerError, "failed to get session")
		}

		// collect and check state to prevent CSRF
		receivedState := c.QueryParam("state")
		expectedState, ok := sess.Values["state"].(string)
		if !ok || receivedState != expectedState {
			log.Debugf("OIDC state mismatch for provider %s", providerId)
			return c.String(http.StatusUnauthorized, "state mismatch")
		}

		code := c.QueryParam("code")
		if code == "" {
			log.Debugf("OIDC code missing from provider %s", providerId)
			return c.String(http.StatusBadRequest, "code parameter missing")
		}

		oauth2Token, err := oidcProv.oauth2Config.Exchange(ctx, code)
		if err != nil {
			log.WithError(err).Error("Failed to get token")
			return c.String(http.StatusInternalServerError, "failed to get token")
		}

		ui, err := oidcProv.provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
		if err != nil {
			log.WithError(err).Error("Failed to get user info")
			return c.String(http.StatusInternalServerError, "failed to get user info")
		}
		var uiClaims map[string]any
		if err := ui.Claims(&uiClaims); err != nil {
			log.WithError(err).Error("Failed to parse user info claims")
			return c.String(http.StatusInternalServerError, "failed to parse user info claims")
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			log.Debugf("OIDC id_token missing from token")
			return c.String(http.StatusInternalServerError, "no id_token in token response")
		}

		verifier := oidcProv.provider.Verifier(&oidc.Config{ClientID: oidcProv.oauth2Config.ClientID})
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			log.WithError(err).Error("Failed to verify token")
			return c.String(http.StatusUnauthorized, "failed to verify ID token")
		}

		// collect claims from id token
		var idTokenClaims jwtClaims
		if err := idToken.Claims(&idTokenClaims); err != nil {
			log.WithError(err).Error("Failed to parse claims")
			return c.String(http.StatusInternalServerError, "failed to parse claims")
		}

		providerSessions, ok := sess.Values[providerSessionsKey].(map[string]ProviderSession)
		if !ok || providerSessions == nil {
			providerSessions = make(map[string]ProviderSession)
		}
		providerSessions[providerId] = ProviderSession{
			ExpiresAt: oauth2Token.Expiry.Unix(),
			Subject:   idTokenClaims.Subject,
			Groups:    idTokenClaims.Groups,
			UserInfo:  uiClaims,
		}
		sess.Values[providerSessionsKey] = providerSessions

		// store original target url
		redirectURL, _ := sess.Values["original_path"].(string)
		if redirectURL == "" {
			redirectURL = "/"
		}
		// save session
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			log.WithError(err).Error("Failed to save session")
			return c.String(http.StatusInternalServerError, "failed to save session")
		}
		// redirect to original target
		return c.Redirect(http.StatusFound, redirectURL)
	}
}

// CreateMiddleware create a middleware, which protect all following routes.
// It checks for user auth and redirect to IdP auth url if needed or redirect to an error page.
// The user must be in one of the allowedGroups to pass the auth test.
// If the list is empty, then the group check is disabled.
// Also, it evaluates the expression if present and only allows access if the expression evaluates to true.
func (o *OIDC) CreateMiddleware(protection *StaticPageProtection) (echo.MiddlewareFunc, error) {
	providerId := protection.Provider
	allowedGroups := protection.Groups

	var expression *Expression = nil
	if protection.Expression != "" {
		expr, err := newExpression(protection.Expression)
		if err != nil {
			log.WithError(err).Error("Error compiling expression")
			return nil, err
		}
		expression = expr
	}

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

			if !ok || (providerSession.ExpiresAt > 0 && providerSession.ExpiresAt < time.Now().Unix()) {
				return redirectForAuth(oauth2Config, c)
			}

			if expression != nil {
				result, err := expression.Eval(providerSession.UserInfo)
				if err != nil {
					log.WithError(err).Error("Error evaluating expression")
					return c.String(http.StatusInternalServerError, "error evaluating access expression")
				}
				if !result {
					return c.String(http.StatusForbidden, "You do not have the required permissions to access this resource.")
				}
			}

			if !checkHasOneGroup(allowedGroups, providerSession.Groups) {
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
		log.WithError(err).Error("Session cannot be retrieved")
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
