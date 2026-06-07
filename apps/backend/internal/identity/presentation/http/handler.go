package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/bowerbird/internal/identity/application"
	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/config"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	authService     *application.AuthService
	identityService *application.IdentityService
	googleConfig    *oauth2.Config
	microsoftConfig *oauth2.Config
	frontendURL     string
}

func NewAuthHandler(
	authService *application.AuthService,
	identityService *application.IdentityService,
	googleConfig *oauth2.Config,
	microsoftConfig *oauth2.Config,
	frontendURL string,
) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		identityService: identityService,
		googleConfig:    googleConfig,
		microsoftConfig: microsoftConfig,
		frontendURL:     frontendURL,
	}
}

func (h *AuthHandler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, cfg config.Config) {
	mux.HandleFunc("POST /api/v1/auth/register-local", api.Wrap(h.RegisterLocal, cfg))
	mux.HandleFunc("POST /api/v1/auth/login-local", api.Wrap(h.LoginLocal, cfg))
	mux.HandleFunc("POST /api/v1/auth/refresh", api.Wrap(h.RefreshToken, cfg))
	mux.HandleFunc("POST /api/v1/auth/logout", api.Wrap(h.Logout, cfg))
	mux.HandleFunc("GET /api/v1/auth/google/login", api.Wrap(h.OAuthGoogleLogin, cfg))
	mux.HandleFunc("GET /api/v1/auth/google/callback", api.Wrap(h.OAuthGoogleCallback, cfg))
	mux.HandleFunc("GET /api/v1/auth/microsoft/login", api.Wrap(h.OAuthMicrosoftLogin, cfg))
	mux.HandleFunc("GET /api/v1/auth/microsoft/callback", api.Wrap(h.OAuthMicrosoftCallback, cfg))

	// Protected routes
	mux.Handle("GET /api/v1/identity/tenants", authMiddleware(api.Wrap(h.ListUserTenants, cfg)))
	mux.Handle("POST /api/v1/identity/tenants/{tenant_id}/leave", authMiddleware(api.Wrap(h.LeaveTenant, cfg)))
	mux.Handle("DELETE /api/v1/identity/account", authMiddleware(api.Wrap(h.DeleteAccount, cfg)))
}

type LocalAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour), // Must match config
	})
}

func (h *AuthHandler) RegisterLocal(w http.ResponseWriter, r *http.Request) error {
	var req LocalAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request")
	}

	tokens, err := h.authService.RegisterLocal(r.Context(), req.Email, req.Password)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to register")
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	return api.Success(w, http.StatusOK, AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

func (h *AuthHandler) LoginLocal(w http.ResponseWriter, r *http.Request) error {
	var req LocalAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request")
	}

	tokens, err := h.authService.LoginLocal(r.Context(), req.Email, req.Password)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeUnauthorized, "invalid credentials")
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	return api.Success(w, http.StatusOK, AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeUnauthorized, "missing refresh token")
	}

	tokens, err := h.authService.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeUnauthorized, "invalid refresh token")
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	return api.Success(w, http.StatusOK, AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *AuthHandler) OAuthGoogleLogin(w http.ResponseWriter, r *http.Request) error {
	slog.Info("Starting Identity Google login flow", "state", "state-token")
	if h.googleConfig == nil {
		return appErrors.New(appErrors.CodeNotImplemented, "google oauth not configured")
	}
	url := h.googleConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthGoogleCallback(w http.ResponseWriter, r *http.Request) error {
	state := r.FormValue("state")
	slog.Info("Received Identity Google login callback", "state", state, "error", r.FormValue("error"))

	redirectOnError := func(reason string) error {
		slog.Error("Identity Google login callback failed", "reason", reason)
		http.Redirect(w, r, h.frontendURL+"/login?error=google_auth_failed", http.StatusTemporaryRedirect)
		return nil
	}

	if h.googleConfig == nil {
		return redirectOnError("google oauth not configured")
	}

	if state != "state-token" {
		return redirectOnError("invalid oauth state")
	}

	code := r.FormValue("code")
	if code == "" {
		return redirectOnError("missing code parameter")
	}

	token, err := h.googleConfig.Exchange(r.Context(), code)
	if err != nil {
		return redirectOnError("code exchange failed")
	}

	client := h.googleConfig.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return redirectOnError("failed getting user info")
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return redirectOnError("failed parsing user info")
	}

	slog.Info("Fetched Identity Google user info", "email", userInfo.Email, "provider_id", userInfo.ID)

	tokens, err := h.authService.OAuthLogin(r.Context(), userInfo.Email, "google", userInfo.ID, userInfo.Name, userInfo.Picture)
	if err != nil {
		return redirectOnError("oauth login failed")
	}

	slog.Info("Identity Google login successful", "email", userInfo.Email)

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	http.Redirect(w, r, h.frontendURL+"/lobby", http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthMicrosoftLogin(w http.ResponseWriter, r *http.Request) error {
	slog.Info("Starting Identity Microsoft login flow", "state", "state-token")
	if h.microsoftConfig == nil {
		return appErrors.New(appErrors.CodeNotImplemented, "microsoft oauth not configured")
	}
	url := h.microsoftConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthMicrosoftCallback(w http.ResponseWriter, r *http.Request) error {
	state := r.FormValue("state")
	slog.Info("Received Identity Microsoft login callback", "state", state, "error", r.FormValue("error"))

	redirectOnError := func(reason string) error {
		slog.Error("Identity Microsoft login callback failed", "reason", reason)
		http.Redirect(w, r, h.frontendURL+"/login?error=microsoft_auth_failed", http.StatusTemporaryRedirect)
		return nil
	}

	if h.microsoftConfig == nil {
		return redirectOnError("microsoft oauth not configured")
	}

	if state != "state-token" {
		return redirectOnError("invalid oauth state")
	}

	code := r.FormValue("code")
	if code == "" {
		return redirectOnError("missing code parameter")
	}

	token, err := h.microsoftConfig.Exchange(r.Context(), code)
	if err != nil {
		return redirectOnError("code exchange failed")
	}

	client := h.microsoftConfig.Client(r.Context(), token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return redirectOnError("failed getting user info")
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"userPrincipalName"`
		Name  string `json:"displayName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return redirectOnError("failed parsing user info")
	}

	slog.Info("Fetched Identity Microsoft user info", "email", userInfo.Email, "provider_id", userInfo.ID)

	tokens, err := h.authService.OAuthLogin(r.Context(), userInfo.Email, "microsoft", userInfo.ID, userInfo.Name, "")
	if err != nil {
		return redirectOnError("oauth login failed")
	}

	slog.Info("Identity Microsoft login successful", "email", userInfo.Email)

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	http.Redirect(w, r, h.frontendURL+"/lobby", http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) ListUserTenants(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	tenants, err := h.identityService.ListUserTenants(r.Context(), claims.UserID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list tenants")
	}

	return api.Success(w, http.StatusOK, tenants)
}

func (h *AuthHandler) LeaveTenant(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		return appErrors.New(appErrors.CodeValidation, "tenant_id is required")
	}

	err := h.identityService.LeaveTenant(r.Context(), claims.UserID, tenantID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to leave tenant")
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	err := h.identityService.DeleteAccount(r.Context(), claims.UserID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to delete account")
	}

	return h.Logout(w, r)
}
