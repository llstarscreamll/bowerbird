package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/identity/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
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

func (h *AuthHandler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/v1/auth/register-local", h.RegisterLocal)
	mux.HandleFunc("POST /api/v1/auth/login-local", h.LoginLocal)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.RefreshToken)
	mux.HandleFunc("POST /api/v1/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/v1/auth/google/login", h.OAuthGoogleLogin)
	mux.HandleFunc("GET /api/v1/auth/google/callback", h.OAuthGoogleCallback)
	mux.HandleFunc("GET /api/v1/auth/microsoft/login", h.OAuthMicrosoftLogin)
	mux.HandleFunc("GET /api/v1/auth/microsoft/callback", h.OAuthMicrosoftCallback)

	// Protected routes
	mux.Handle("GET /api/v1/identity/tenants", authMiddleware(http.HandlerFunc(h.ListUserTenants)))
	mux.Handle("POST /api/v1/identity/tenants/{tenant_id}/leave", authMiddleware(http.HandlerFunc(h.LeaveTenant)))
	mux.Handle("DELETE /api/v1/identity/account", authMiddleware(http.HandlerFunc(h.DeleteAccount)))
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

func (h *AuthHandler) RegisterLocal(w http.ResponseWriter, r *http.Request) {
	var req LocalAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	tokens, err := h.authService.RegisterLocal(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

func (h *AuthHandler) LoginLocal(w http.ResponseWriter, r *http.Request) {
	var req LocalAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	tokens, err := h.authService.LoginLocal(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	tokens, err := h.authService.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
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
}

func (h *AuthHandler) OAuthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if h.googleConfig == nil {
		http.Error(w, "google oauth not configured", http.StatusNotImplemented)
		return
	}
	// Note: in a real app, generate a secure state and store it in a cookie
	url := h.googleConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) OAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if h.googleConfig == nil {
		http.Error(w, "google oauth not configured", http.StatusNotImplemented)
		return
	}

	state := r.FormValue("state")
	if state != "state-token" {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := h.googleConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "code exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := h.googleConfig.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "failed getting user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "failed parsing user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tokens, err := h.authService.OAuthLogin(r.Context(), userInfo.Email, "google", userInfo.ID, userInfo.Name, userInfo.Picture)
	if err != nil {
		http.Error(w, "oauth login failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)

	// Redirect to frontend with access token in fragment (or via a specific callback page)
	http.Redirect(w, r, h.frontendURL+"/auth/callback?access_token="+tokens.AccessToken, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) OAuthMicrosoftLogin(w http.ResponseWriter, r *http.Request) {
	if h.microsoftConfig == nil {
		http.Error(w, "microsoft oauth not configured", http.StatusNotImplemented)
		return
	}
	url := h.microsoftConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) OAuthMicrosoftCallback(w http.ResponseWriter, r *http.Request) {
	if h.microsoftConfig == nil {
		http.Error(w, "microsoft oauth not configured", http.StatusNotImplemented)
		return
	}

	state := r.FormValue("state")
	if state != "state-token" {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := h.microsoftConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "code exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := h.microsoftConfig.Client(r.Context(), token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		http.Error(w, "failed getting user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"userPrincipalName"`
		Name  string `json:"displayName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "failed parsing user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tokens, err := h.authService.OAuthLogin(r.Context(), userInfo.Email, "microsoft", userInfo.ID, userInfo.Name, "")
	if err != nil {
		http.Error(w, "oauth login failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken)
	http.Redirect(w, r, h.frontendURL+"/auth/callback?access_token="+tokens.AccessToken, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) ListUserTenants(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tenants, err := h.identityService.ListUserTenants(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

func (h *AuthHandler) LeaveTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	err := h.identityService.LeaveTenant(r.Context(), claims.UserID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.identityService.DeleteAccount(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Logout(w, r)
}
