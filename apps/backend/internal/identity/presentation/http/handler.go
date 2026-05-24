package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/identity/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
)

type AuthHandler struct {
	authService     *application.AuthService
	identityService *application.IdentityService
}

func NewAuthHandler(authService *application.AuthService, identityService *application.IdentityService) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		identityService: identityService,
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
	http.Error(w, "not implemented fully yet", http.StatusNotImplemented)
}

func (h *AuthHandler) OAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented fully yet", http.StatusNotImplemented)
}

func (h *AuthHandler) OAuthMicrosoftLogin(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented fully yet", http.StatusNotImplemented)
}

func (h *AuthHandler) OAuthMicrosoftCallback(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented fully yet", http.StatusNotImplemented)
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
