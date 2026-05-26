package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/identity/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
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

func (h *AuthHandler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.HandleFunc("POST /api/v1/auth/register-local", api.Wrap(h.RegisterLocal, isDev))
	mux.HandleFunc("POST /api/v1/auth/login-local", api.Wrap(h.LoginLocal, isDev))
	mux.HandleFunc("POST /api/v1/auth/refresh", api.Wrap(h.RefreshToken, isDev))
	mux.HandleFunc("POST /api/v1/auth/logout", api.Wrap(h.Logout, isDev))
	mux.HandleFunc("GET /api/v1/auth/google/login", api.Wrap(h.OAuthGoogleLogin, isDev))
	mux.HandleFunc("GET /api/v1/auth/google/callback", api.Wrap(h.OAuthGoogleCallback, isDev))
	mux.HandleFunc("GET /api/v1/auth/microsoft/login", api.Wrap(h.OAuthMicrosoftLogin, isDev))
	mux.HandleFunc("GET /api/v1/auth/microsoft/callback", api.Wrap(h.OAuthMicrosoftCallback, isDev))

	// Protected routes
	mux.Handle("GET /api/v1/identity/tenants", authMiddleware(api.Wrap(h.ListUserTenants, isDev)))
	mux.Handle("POST /api/v1/identity/tenants/{tenant_id}/leave", authMiddleware(api.Wrap(h.LeaveTenant, isDev)))
	mux.Handle("DELETE /api/v1/identity/account", authMiddleware(api.Wrap(h.DeleteAccount, isDev)))
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
		return apperrors.Wrap(err, apperrors.CodeValidation, "invalid request")
	}

	tokens, err := h.authService.RegisterLocal(r.Context(), req.Email, req.Password)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to register")
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
		return apperrors.Wrap(err, apperrors.CodeValidation, "invalid request")
	}

	tokens, err := h.authService.LoginLocal(r.Context(), req.Email, req.Password)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeUnauthorized, "invalid credentials")
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
		return apperrors.Wrap(err, apperrors.CodeUnauthorized, "missing refresh token")
	}

	tokens, err := h.authService.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeUnauthorized, "invalid refresh token")
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
	if h.googleConfig == nil {
		return apperrors.New(apperrors.CodeNotImplemented, "google oauth not configured")
	}
	url := h.googleConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthGoogleCallback(w http.ResponseWriter, r *http.Request) error {
	if h.googleConfig == nil {
		return apperrors.New(apperrors.CodeNotImplemented, "google oauth not configured")
	}

	state := r.FormValue("state")
	if state != "state-token" {
		return apperrors.New(apperrors.CodeValidation, "invalid oauth state")
	}

	code := r.FormValue("code")
	token, err := h.googleConfig.Exchange(r.Context(), code)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "code exchange failed")
	}

	client := h.googleConfig.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed getting user info")
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed parsing user info")
	}

	tokens, err := h.authService.OAuthLogin(r.Context(), userInfo.Email, "google", userInfo.ID, userInfo.Name, userInfo.Picture)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "oauth login failed")
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	http.Redirect(w, r, h.frontendURL+"/lobby", http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthMicrosoftLogin(w http.ResponseWriter, r *http.Request) error {
	if h.microsoftConfig == nil {
		return apperrors.New(apperrors.CodeNotImplemented, "microsoft oauth not configured")
	}
	url := h.microsoftConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) OAuthMicrosoftCallback(w http.ResponseWriter, r *http.Request) error {
	if h.microsoftConfig == nil {
		return apperrors.New(apperrors.CodeNotImplemented, "microsoft oauth not configured")
	}

	state := r.FormValue("state")
	if state != "state-token" {
		return apperrors.New(apperrors.CodeValidation, "invalid oauth state")
	}

	code := r.FormValue("code")
	token, err := h.microsoftConfig.Exchange(r.Context(), code)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "code exchange failed")
	}

	client := h.microsoftConfig.Client(r.Context(), token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed getting user info")
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"userPrincipalName"`
		Name  string `json:"displayName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed parsing user info")
	}

	tokens, err := h.authService.OAuthLogin(r.Context(), userInfo.Email, "microsoft", userInfo.ID, userInfo.Name, "")
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "oauth login failed")
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	http.Redirect(w, r, h.frontendURL+"/lobby", http.StatusTemporaryRedirect)
	return nil
}

func (h *AuthHandler) ListUserTenants(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return apperrors.New(apperrors.CodeUnauthorized, "unauthorized")
	}

	tenants, err := h.identityService.ListUserTenants(r.Context(), claims.UserID)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to list tenants")
	}

	return api.Success(w, http.StatusOK, tenants)
}

func (h *AuthHandler) LeaveTenant(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return apperrors.New(apperrors.CodeUnauthorized, "unauthorized")
	}

	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		return apperrors.New(apperrors.CodeValidation, "tenant_id is required")
	}

	err := h.identityService.LeaveTenant(r.Context(), claims.UserID, tenantID)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to leave tenant")
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return apperrors.New(apperrors.CodeUnauthorized, "unauthorized")
	}

	err := h.identityService.DeleteAccount(r.Context(), claims.UserID)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to delete account")
	}

	return h.Logout(w, r)
}
