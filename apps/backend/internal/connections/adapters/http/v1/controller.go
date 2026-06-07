package v1

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

	"github.com/bowerbird/internal/connections/domain"
	"github.com/bowerbird/internal/platform/auth"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/tenant"
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

type Controller struct {
	repo         domain.Repository
	credSetter   ConnectionCredentialsSetter
	googleConfig *oauth2.Config
	tokenGen     TokenValidator
	stateProtect StateProtector
	publisher    EventPublisher
	frontendURL  string
}

func NewController(repo domain.Repository, credSetter ConnectionCredentialsSetter, googleConfig *oauth2.Config, tokenGen TokenValidator, stateProtect StateProtector, publisher EventPublisher, frontendURL string) *Controller {
	if repo == nil {
		panic("connections repository is required")
	}

	if tokenGen == nil {
		panic("token validator is required")
	}

	return &Controller{
		repo:         repo,
		credSetter:   credSetter,
		googleConfig: googleConfig,
		tokenGen:     tokenGen,
		stateProtect: stateProtect,
		publisher:    publisher,
		frontendURL:  frontendURL,
	}
}

func (c *Controller) ListConnections(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	connections, err := c.repo.ListAll(ctx)
	if err != nil {
		return err
	}

	response := make([]connectionResponse, 0, len(connections))
	for _, connection := range connections {
		response = append(response, newConnectionResponse(connection))
	}

	return api.Success(w, http.StatusOK, map[string]interface{}{"data": response})
}

func (c *Controller) DeleteConnection(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	connectionID := r.PathValue("id")
	if connectionID == "" {
		return appErrors.New(appErrors.CodeValidation, "connection id is required")
	}

	err := c.repo.Delete(ctx, connectionID)
	if err != nil {
		return err
	}

	return api.Success(w, http.StatusNoContent, nil)
}

func (c *Controller) GoogleConnect(w http.ResponseWriter, r *http.Request) error {
	if c.googleConfig == nil {
		return appErrors.New(appErrors.CodeInternal, "google integration not configured")
	}

	authHeader := r.Header.Get("Authorization")
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return appErrors.New(appErrors.CodeUnauthorized, "missing or invalid authorization header")
	}
	tokenString := parts[1]

	claims, err := c.tokenGen.ValidateAccessToken(tokenString)
	if err != nil {
		return appErrors.New(appErrors.CodeUnauthorized, "invalid authorization token")
	}

	tenantID, err := tenant.TenantIDFromContext(r.Context())
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, "missing tenant context")
	}

	statePayload := fmt.Sprintf("%s|%s|%d", claims.Subject, tenantID, time.Now().Unix())
	encryptedState, err := c.stateProtect.Encrypt([]byte(statePayload))
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to secure state parameter")
	}
	opaqueState := base64.URLEncoding.EncodeToString(encryptedState)

	slog.Info("Starting Google connection flow", "user_id", claims.Subject, "tenant_id", tenantID, "state", opaqueState)

	url := c.googleConfig.AuthCodeURL(opaqueState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return api.Success(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"auth_url": url},
	})
}

func (c *Controller) GoogleCallback(w http.ResponseWriter, r *http.Request) error {
	stateParam := r.URL.Query().Get("state")
	slog.Info("Received Google connection callback", "state", stateParam, "error", r.URL.Query().Get("error"))

	redirectOnError := func(tenantID string, reason string) error {
		slog.Error("Google connection callback failed", "tenant_id", tenantID, "reason", reason)
		frontendURL := c.frontendURL
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

	if c.googleConfig == nil {
		return redirectOnError("", "google config is nil")
	}

	if stateParam == "" {
		return redirectOnError("", "missing state parameter")
	}

	encryptedState, err := base64.URLEncoding.DecodeString(stateParam)
	if err != nil {
		return redirectOnError("", "failed to decode state base64")
	}

	decryptedState, err := c.stateProtect.Decrypt(encryptedState)
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

	if time.Since(time.Unix(timestamp, 0)) > 10*time.Minute {
		return redirectOnError(tenantID, "state expired")
	}

	slog.Info("Decrypted Google connection state", "user_id", userID, "tenant_id", tenantID)

	ctx := r.Context()
	code := r.URL.Query().Get("code")
	if code == "" {
		return redirectOnError(tenantID, "missing code parameter")
	}

	token, err := c.googleConfig.Exchange(ctx, code)
	if err != nil {
		return redirectOnError(tenantID, fmt.Sprintf("exchange token failed: %v", err))
	}

	client := c.googleConfig.Client(ctx, token)
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

	connection := &domain.Connection{
		ID:                   id.NewULID(),
		OwnerUserID:          userID,
		Provider:             "gmail",
		ProviderAccountEmail: userInfo.Email,
		Status:               domain.ConnectionStatusActive,
		GrantedScopes:        c.googleConfig.Scopes,
		SharingPolicy:        domain.SharingPolicyPrivate,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := c.credSetter.SetEncryptedCredentials(connection, tokenBytes); err != nil {
		return redirectOnError(tenantID, fmt.Sprintf("encrypt credentials failed: %v", err))
	}

	ctx = tenant.WithTenantID(ctx, tenantID)

	if err := c.repo.Upsert(ctx, connection); err != nil {
		return redirectOnError(tenantID, fmt.Sprintf("save connection failed: %v", err))
	}

	slog.Info("Google connection saved successfully", "connection_id", connection.ID, "email", userInfo.Email, "user_id", userID, "tenant_id", tenantID)

	if c.publisher != nil {
		if err := c.publisher.PublishConnectionAdded(ctx, connection); err != nil {
			slog.Error("failed to publish ConnectionAdded event", "error", err, "connection_id", connection.ID)
		}
	}

	slog.Info("ConnectionAdded event published successfully", "connection_id", connection.ID)

	frontendURL := c.frontendURL
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
