package v1

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bowerbird/internal/identity/application"
	"github.com/bowerbird/internal/identity/application/commands"
	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/config"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
	"github.com/bowerbird/internal/platform/http/ratelimit"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	cmds            application.Commands
	identityService *application.IdentityService
	googleConfig    *oauth2.Config
	microsoftConfig *oauth2.Config
	frontendURL     string
	refreshTTL      time.Duration
	authLimiter     *ratelimit.Limiter
}

func NewAuthHandler(
	cmds application.Commands,
	identityService *application.IdentityService,
	googleConfig *oauth2.Config,
	microsoftConfig *oauth2.Config,
	frontendURL string,
	refreshTTL time.Duration,
) *AuthHandler {
	return &AuthHandler{
		cmds:            cmds,
		identityService: identityService,
		googleConfig:    googleConfig,
		microsoftConfig: microsoftConfig,
		frontendURL:     frontendURL,
		refreshTTL:      refreshTTL,
		authLimiter:     ratelimit.New(20, time.Minute),
	}
}

func (h *AuthHandler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, cfg config.Config) {
	mux.HandleFunc("POST /api/v1/auth/register-local", api.Wrap(h.authLimiter.Protect(h.RegisterLocal), cfg))
	mux.HandleFunc("POST /api/v1/auth/login-local", api.Wrap(h.authLimiter.Protect(h.LoginLocal), cfg))
	mux.HandleFunc("POST /api/v1/auth/refresh", api.Wrap(h.authLimiter.Protect(h.RefreshToken), cfg))
	mux.HandleFunc("POST /api/v1/auth/logout", api.Wrap(h.Logout, cfg))
	mux.HandleFunc("GET /api/v1/auth/google/login", api.Wrap(h.OAuthGoogleLogin, cfg))
	mux.HandleFunc("GET /api/v1/auth/google/callback", api.Wrap(h.OAuthGoogleCallback, cfg))
	mux.HandleFunc("GET /api/v1/auth/microsoft/login", api.Wrap(h.OAuthMicrosoftLogin, cfg))
	mux.HandleFunc("GET /api/v1/auth/microsoft/callback", api.Wrap(h.OAuthMicrosoftCallback, cfg))

	mux.Handle("GET /api/v1/identity/me", authMiddleware(api.Wrap(h.GetMe, cfg)))
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
	ttl := h.refreshTTL
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/v1/auth",
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func (h *AuthHandler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, commands.ErrInvalidCredentials):
		return appErrors.Wrap(err, appErrors.CodeUnauthorized, "invalid credentials")
	case errors.Is(err, commands.ErrLocalAuthDisabled):
		return appErrors.Wrap(err, appErrors.CodeForbidden, "local auth is disabled")
	case errors.Is(err, commands.ErrEmailNotVerified), errors.Is(err, commands.ErrInvalidEmail):
		return appErrors.Wrap(err, appErrors.CodeValidation, err.Error())
	case errors.Is(err, auth.ErrPasswordTooShort), errors.Is(err, auth.ErrPasswordTooLong):
		return appErrors.Wrap(err, appErrors.CodeValidation, err.Error())
	case errors.Is(err, auth.ErrRefreshNotFound), errors.Is(err, auth.ErrRefreshRevoked),
		errors.Is(err, auth.ErrRefreshExpired), errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrExpiredToken):
		return appErrors.Wrap(err, appErrors.CodeUnauthorized, "invalid refresh token")
	default:
		return appErrors.Wrap(err, appErrors.CodeInternal, "authentication failed")
	}
}

func (h *AuthHandler) RegisterLocal(w http.ResponseWriter, r *http.Request) error {
	var req LocalAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request")
	}

	tokens, err := h.cmds.RegisterLocal.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		return mapAuthError(err)
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

	tokens, err := h.cmds.LoginLocal.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		return mapAuthError(err)
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

	tokens, err := h.cmds.RefreshToken.Execute(r.Context(), cookie.Value)
	if err != nil {
		h.clearRefreshTokenCookie(w)
		return mapAuthError(err)
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	return api.Success(w, http.StatusOK, AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		_ = h.cmds.RevokeRefreshToken.Execute(r.Context(), cookie.Value)
	}
	h.clearRefreshTokenCookie(w)
	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *AuthHandler) OAuthGoogleLogin(w http.ResponseWriter, r *http.Request) error {
	if h.googleConfig == nil {
		return appErrors.New(appErrors.CodeNotImplemented, "google oauth not configured")
	}
	state, err := auth.NewOAuthState(w)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to start oauth")
	}
	url := h.googleConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthGoogleCallback(w http.ResponseWriter, r *http.Request) error {
	state := r.FormValue("state")
	slog.Info("Received Identity Google login callback", "error", r.FormValue("error"))

	redirectOnError := func(reason string, cause error) error {
		if cause != nil {
			slog.Error("Identity Google login callback failed", "reason", reason, "error", cause)
		} else {
			slog.Error("Identity Google login callback failed", "reason", reason)
		}
		http.Redirect(w, r, h.frontendURL+"/login?error=google_auth_failed", http.StatusTemporaryRedirect)
		return nil
	}

	if h.googleConfig == nil {
		return redirectOnError("google oauth not configured", nil)
	}
	if !auth.ConsumeOAuthState(w, r, state) {
		return redirectOnError("invalid oauth state", nil)
	}

	code := r.FormValue("code")
	if code == "" {
		return redirectOnError("missing code parameter", nil)
	}

	token, err := h.googleConfig.Exchange(r.Context(), code)
	if err != nil {
		return redirectOnError("code exchange failed", err)
	}

	client := h.googleConfig.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return redirectOnError("failed getting user info", err)
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return redirectOnError("failed parsing user info", err)
	}

	tokens, err := h.cmds.OAuthLogin.Execute(
		r.Context(),
		userInfo.Email,
		"google",
		userInfo.ID,
		userInfo.Name,
		userInfo.Picture,
		userInfo.VerifiedEmail,
	)
	if err != nil {
		return redirectOnError("oauth login failed", err)
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	http.Redirect(w, r, h.frontendURL+"/lobby", http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthMicrosoftLogin(w http.ResponseWriter, r *http.Request) error {
	if h.microsoftConfig == nil {
		return appErrors.New(appErrors.CodeNotImplemented, "microsoft oauth not configured")
	}
	state, err := auth.NewOAuthState(w)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to start oauth")
	}
	url := h.microsoftConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthMicrosoftCallback(w http.ResponseWriter, r *http.Request) error {
	state := r.FormValue("state")
	slog.Info("Received Identity Microsoft login callback", "error", r.FormValue("error"))

	redirectOnError := func(reason string, cause error) error {
		if cause != nil {
			slog.Error("Identity Microsoft login callback failed", "reason", reason, "error", cause)
		} else {
			slog.Error("Identity Microsoft login callback failed", "reason", reason)
		}
		http.Redirect(w, r, h.frontendURL+"/login?error=microsoft_auth_failed", http.StatusTemporaryRedirect)
		return nil
	}

	if h.microsoftConfig == nil {
		return redirectOnError("microsoft oauth not configured", nil)
	}
	if !auth.ConsumeOAuthState(w, r, state) {
		return redirectOnError("invalid oauth state", nil)
	}

	code := r.FormValue("code")
	if code == "" {
		return redirectOnError("missing code parameter", nil)
	}

	token, err := h.microsoftConfig.Exchange(r.Context(), code)
	if err != nil {
		return redirectOnError("code exchange failed", err)
	}

	client := h.microsoftConfig.Client(r.Context(), token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me?$select=id,displayName,mail,userPrincipalName")
	if err != nil {
		return redirectOnError("failed getting user info", err)
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID                string `json:"id"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
		Name              string `json:"displayName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return redirectOnError("failed parsing user info", err)
	}

	email, verified := microsoftEmail(userInfo.Mail, userInfo.UserPrincipalName)
	tokens, err := h.cmds.OAuthLogin.Execute(r.Context(), email, "microsoft", userInfo.ID, userInfo.Name, "", verified)
	if err != nil {
		return redirectOnError("oauth login failed", err)
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	http.Redirect(w, r, h.frontendURL+"/lobby", http.StatusTemporaryRedirect)
	return nil
}

func microsoftEmail(mail, upn string) (email string, verified bool) {
	mail = strings.TrimSpace(mail)
	upn = strings.TrimSpace(upn)
	if mail != "" && strings.Contains(mail, "@") {
		return mail, true
	}
	// Prefer mail; UPN is accepted only when it looks like an email (typical for Microsoft accounts).
	if strings.Contains(upn, "@") {
		return upn, true
	}
	return "", false
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	me, err := h.identityService.GetMe(r.Context(), claims.UserID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to load profile")
	}

	return api.Success(w, http.StatusOK, me)
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

	_ = h.cmds.RevokeAllRefreshTokens.Execute(r.Context(), claims.UserID)

	err := h.identityService.DeleteAccount(r.Context(), claims.UserID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to delete account")
	}

	return h.Logout(w, r)
}
