package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/connections/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/id"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
	"golang.org/x/oauth2"
)

type ConnectionCredentialsSetter interface {
	SetEncryptedCredentials(account *domain.Connection, plaintext []byte) error
}

type TokenValidator interface {
	ValidateAccessToken(tokenString string) (*auth.CustomClaims, error)
}

type StateProtector interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type EventPublisher interface {
	PublishConnectionAdded(ctx context.Context, connection *domain.Connection) error
}

type Handler struct {
	repo         domain.Repository
	credSetter   ConnectionCredentialsSetter
	googleConfig *oauth2.Config
	tokenGen     TokenValidator
	stateProtect StateProtector
	publisher    EventPublisher
	frontendURL  string
}

func NewHandler(repo domain.Repository, credSetter ConnectionCredentialsSetter, googleConfig *oauth2.Config, tokenGen TokenValidator, stateProtect StateProtector, publisher EventPublisher, frontendURL string) *Handler {
	return &Handler{
		repo:         repo,
		credSetter:   credSetter,
		googleConfig: googleConfig,
		tokenGen:     tokenGen,
		stateProtect: stateProtect,
		publisher:    publisher,
		frontendURL:  frontendURL,
	}
}

func (h *Handler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.Handle("GET /api/v1/connections", authMiddleware(api.Wrap(h.ListConnections, isDev)))
	mux.Handle("GET /api/v1/connections/google", authMiddleware(api.Wrap(h.GoogleConnect, isDev)))
	// Callback doesn't use authMiddleware because it's a browser redirect, we'll validate the state param.
	mux.Handle("GET /api/v1/connections/google/callback", api.Wrap(h.GoogleCallback, isDev))
	mux.Handle("DELETE /api/v1/connections/{id}", authMiddleware(api.Wrap(h.DeleteConnection, isDev)))
}

type connectionResponse struct {
	ID                   string `json:"id"`
	Provider             string `json:"provider"`
	ProviderAccountEmail string `json:"provider_account_email"`
	Status               string `json:"status"`
	SharingPolicy        string `json:"sharing_policy"`
}

func (h *Handler) ListConnections(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	connections, err := h.repo.ListAll(ctx)
	if err != nil {
		return err
	}

	response := make([]connectionResponse, 0, len(connections))
	for _, c := range connections {
		response = append(response, connectionResponse{
			ID:                   c.ID,
			Provider:             c.Provider,
			ProviderAccountEmail: c.ProviderAccountEmail,
			Status:               c.Status,
			SharingPolicy:        c.SharingPolicy,
		})
	}

	return api.Success(w, http.StatusOK, map[string]interface{}{"data": response})
}

func (h *Handler) DeleteConnection(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		return apperrors.New(apperrors.CodeValidation, "connection id is required")
	}

	err := h.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	return api.Success(w, http.StatusNoContent, nil)
}

func (h *Handler) GoogleConnect(w http.ResponseWriter, r *http.Request) error {
	if h.googleConfig == nil {
		return apperrors.New(apperrors.CodeInternal, "google integration not configured")
	}

	authHeader := r.Header.Get("Authorization")
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return apperrors.New(apperrors.CodeUnauthorized, "missing or invalid authorization header")
	}
	tokenString := parts[1]

	claims, err := h.tokenGen.ValidateAccessToken(tokenString)
	if err != nil {
		return apperrors.New(apperrors.CodeUnauthorized, "invalid authorization token")
	}

	tenantID, err := tenant.TenantSlugFromContext(r.Context())
	if err != nil {
		return apperrors.New(apperrors.CodeValidation, "missing tenant context")
	}

	// Create an opaque state containing the UserID, tenantID, and a timestamp
	statePayload := fmt.Sprintf("%s|%s|%d", claims.Subject, tenantID, time.Now().Unix())
	encryptedState, err := h.stateProtect.Encrypt([]byte(statePayload))
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to secure state parameter")
	}
	opaqueState := base64.URLEncoding.EncodeToString(encryptedState)

	slog.Info("Starting Google connection flow", "user_id", claims.Subject, "tenant_id", tenantID, "state", opaqueState)

	url := h.googleConfig.AuthCodeURL(opaqueState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return api.Success(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"auth_url": url},
	})
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) error {
	stateParam := r.URL.Query().Get("state")
	slog.Info("Received Google connection callback", "state", stateParam, "error", r.URL.Query().Get("error"))

	redirectOnError := func(tenantID string, reason string) error {
		slog.Error("Google connection callback failed", "tenant_id", tenantID, "reason", reason)
		frontendURL := h.frontendURL
		if frontendURL == "" {
			frontendURL = "https://app.bowerbird.dev"
		}
		path := "/"
		if tenantID != "" {
			path = "/" + tenantID + "/connections"
		}
		http.Redirect(w, r, frontendURL+path, http.StatusTemporaryRedirect)
		return nil
	}

	if h.googleConfig == nil {
		return redirectOnError("", "google config is nil")
	}

	if stateParam == "" {
		return redirectOnError("", "missing state parameter")
	}

	encryptedState, err := base64.URLEncoding.DecodeString(stateParam)
	if err != nil {
		return redirectOnError("", "failed to decode state base64")
	}

	decryptedState, err := h.stateProtect.Decrypt(encryptedState)
	if err != nil {
		return redirectOnError("", "failed to decrypt state")
	}

	stateParts := strings.Split(string(decryptedState), "|")
	if len(stateParts) != 3 {
		return redirectOnError("", "invalid state format")
	}

	userID := stateParts[0]
	tenantID := stateParts[1]
	timestamp, err := strconv.ParseInt(stateParts[2], 10, 64)
	if err != nil {
		return redirectOnError(tenantID, "invalid state timestamp")
	}

	// State expires in 10 minutes to prevent replay attacks over long periods
	if time.Since(time.Unix(timestamp, 0)) > 10*time.Minute {
		return redirectOnError(tenantID, "state expired")
	}

	slog.Info("Decrypted Google connection state", "user_id", userID, "tenant_id", tenantID)

	ctx := r.Context()
	code := r.URL.Query().Get("code")
	if code == "" {
		return redirectOnError(tenantID, "missing code parameter")
	}

	token, err := h.googleConfig.Exchange(ctx, code)
	if err != nil {
		return redirectOnError(tenantID, fmt.Sprintf("exchange token failed: %v", err))
	}

	// Fetch user email using token
	client := h.googleConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return redirectOnError(tenantID, fmt.Sprintf("fetch user info failed: %v", err))
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return redirectOnError(tenantID, fmt.Sprintf("decode user info failed: %v", err))
	}

	slog.Info("Fetched Google user info", "email", userInfo.Email, "user_id", userID, "tenant_id", tenantID)

	tokenBytes, _ := json.Marshal(token)

	// Create new connection
	conn := &domain.Connection{
		ID:                   id.NewULID(),
		OwnerUserID:          userID,
		Provider:             "gmail",
		ProviderAccountEmail: userInfo.Email,
		Status:               domain.ConnectionStatusActive,
		GrantedScopes:        h.googleConfig.Scopes,
		SharingPolicy:        domain.SharingPolicyPrivate, // Default to private
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := h.credSetter.SetEncryptedCredentials(conn, tokenBytes); err != nil {
		return redirectOnError(tenantID, fmt.Sprintf("encrypt credentials failed: %v", err))
	}

	// Inject tenant ID back into context for the repository
	ctx = tenant.WithTenantSlug(ctx, tenantID)

	if err := h.repo.Upsert(ctx, conn); err != nil {
		return redirectOnError(tenantID, fmt.Sprintf("save connection failed: %v", err))
	}

	slog.Info("Google connection saved successfully", "connection_id", conn.ID, "email", userInfo.Email, "user_id", userID, "tenant_id", tenantID)

	if h.publisher != nil {
		if err := h.publisher.PublishConnectionAdded(ctx, conn); err != nil {
			// Just log the error, don't fail the request since connection was saved
			slog.Error("failed to publish ConnectionAdded event", "error", err, "connection_id", conn.ID)
		}
	}

	slog.Info("ConnectionAdded event published successfully", "connection_id", conn.ID)

	// redirect back to frontend on success
	frontendURL := h.frontendURL
	if frontendURL == "" {
		frontendURL = "https://app.bowerbird.dev"
	}
	path := "/"
	if tenantID != "" {
		path = "/" + tenantID + "/connections"
	}
	http.Redirect(w, r, frontendURL+path, http.StatusTemporaryRedirect)
	return nil
}
