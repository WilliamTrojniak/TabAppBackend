package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/WilliamTrojniak/TabAppBackend/env"
	"github.com/WilliamTrojniak/TabAppBackend/services"
	"github.com/WilliamTrojniak/TabAppBackend/services/sessions"
	"github.com/WilliamTrojniak/TabAppBackend/types"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type CreateUserFn func(context context.Context, user *types.UserCreate) error

type Handler struct {
	logger         *slog.Logger
	handleError    services.HTTPErrorHandler
	createUser     CreateUserFn
	config         oauth2.Config
	provider       *oidc.Provider
	sessionManager *sessions.SessionManager
}

func NewHandler(handleError services.HTTPErrorHandler, sessionManager *sessions.SessionManager, logger *slog.Logger) (*Handler, error) {
	provider, err := oidc.NewProvider(context.TODO(), "https://accounts.google.com")
	if err != nil {
		return nil, err
	}
	config := oauth2.Config{
		ClientID:     env.Envs.OAUTH2_GOOGLE_CLIENT_ID,
		ClientSecret: env.Envs.OAUTH2_GOOGLE_CLIENT_SECRET,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  "http://127.0.0.1:3000/auth/google/callback",
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &Handler{
		handleError:    handleError,
		logger:         logger,
		provider:       provider,
		config:         config,
		sessionManager: sessionManager,
	}, nil
}

func (h *Handler) SetCreateUserFn(fn CreateUserFn) {
	h.createUser = fn
}

func (h *Handler) beginAuthorize(w http.ResponseWriter, r *http.Request) error {
	state, err := randString(16)
	if err != nil {
		return err
	}
	nonce, err := randString(16)
	if err != nil {
		return err
	}

	setCallbackCookie(w, r, "state", state)
	setCallbackCookie(w, r, "nonce", nonce)
	http.Redirect(w, r, h.config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)

	return nil
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) error {
	// Check that the CSRF token matches
	state, err := r.Cookie("state")
	if err != nil {
		return services.NewInternalServiceError(err)
	}

	if r.URL.Query().Get("state") != state.Value {
		return fmt.Errorf("State did not match")
	}

	// Exchange the code for a token
	oauth2Token, err := h.config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		return services.NewInternalServiceError(err)
	}

	// Get the id token from the JWT
	rawIdToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return fmt.Errorf("No id_token field in oauth2 token")
	}

	// Verify the id token
	oidcConfig := &oidc.Config{ClientID: env.Envs.OAUTH2_GOOGLE_CLIENT_ID}
	verifier := h.provider.Verifier(oidcConfig)

	idToken, err := verifier.Verify(r.Context(), rawIdToken)
	if err != nil {
		return services.NewInternalServiceError(err)
	}

	nonce, err := r.Cookie("nonce")
	if err != nil {
		return services.NewInternalServiceError(err)
	}

	if idToken.Nonce != nonce.Value {
		return fmt.Errorf("nonce did not match")
	}

	// Get user data from the id token
	var claims struct {
		Email         string `json:"email"`
		Name          string `json:"name"`
		Sub           string `json:"sub"`
		EmailVerified bool   `json:"email_verified"`
	}

	userInfo, err := h.provider.UserInfo(r.Context(), oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		return services.NewInternalServiceError(err)
	}

	if err := userInfo.Claims(&claims); err != nil {
		return services.NewInternalServiceError(err)
	}

	// Add the user to the database if not already
	err = h.createUser(r.Context(), &types.UserCreate{Id: claims.Sub, Email: claims.Email, Name: claims.Name})
	if err != nil {
		return services.NewInternalServiceError(err)
	}

	_, err = h.sessionManager.CreateSession(w, r, claims.Sub)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {

	err := h.sessionManager.ClearSession(w, r)
	if err != nil {
		return err
	}

	return nil
}

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCallbackCookie(w http.ResponseWriter, r *http.Request, name string, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, c)
}
