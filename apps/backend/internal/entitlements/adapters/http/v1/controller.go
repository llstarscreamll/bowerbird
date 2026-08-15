package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bowerbird/internal/entitlements/application"
	"github.com/bowerbird/internal/entitlements/domain"
	"github.com/bowerbird/internal/platform/auth"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
	"github.com/bowerbird/internal/platform/tenant"
)

type Controller struct {
	svc     *application.Service
	tenants application.TenantDirectory
}

func NewController(svc *application.Service, tenants application.TenantDirectory) *Controller {
	if svc == nil {
		panic("entitlements service is required")
	}
	if tenants == nil {
		panic("tenant directory is required")
	}
	return &Controller{svc: svc, tenants: tenants}
}

type effectiveAccessResponse struct {
	Features []string `json:"features"`
}

type grantResponse struct {
	FeatureKey string  `json:"feature_key"`
	Status     string  `json:"status"`
	Source     string  `json:"source"`
	StartsAt   string  `json:"starts_at"`
	EndsAt     *string `json:"ends_at,omitempty"`
}

type tenantEntitlementsResponse struct {
	TenantID string          `json:"tenant_id"`
	Features []string        `json:"features"`
	Grants   []grantResponse `json:"grants"`
}

type platformTenantResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

type setAccessRequest struct {
	Product string  `json:"product"`
	Feature string  `json:"feature"`
	Enabled bool    `json:"enabled"`
	EndsAt  *string `json:"ends_at"`
}

func (c *Controller) GetEntitlements(w http.ResponseWriter, r *http.Request) error {
	tenantID, err := tenant.TenantIDFromContext(r.Context())
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, "missing tenant context")
	}

	access, err := c.svc.Evaluate(r.Context(), tenantID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to evaluate entitlements")
	}
	return api.Success(w, http.StatusOK, effectiveAccessResponse{Features: access.Features})
}

func (c *Controller) ListPlatformTenants(w http.ResponseWriter, r *http.Request) error {
	tenants, err := c.tenants.ListTenants(r.Context())
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list tenants")
	}

	response := make([]platformTenantResponse, 0, len(tenants))
	for _, item := range tenants {
		response = append(response, platformTenantResponse{
			ID:     item.ID,
			Name:   item.Name,
			Slug:   item.Slug,
			Status: item.Status,
		})
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": response})
}

func (c *Controller) GetPlatformTenantEntitlements(w http.ResponseWriter, r *http.Request) error {
	tenantID := r.PathValue("id")
	if tenantID == "" {
		return appErrors.New(appErrors.CodeValidation, "tenant id is required")
	}
	if err := c.requireTenant(r, tenantID); err != nil {
		return err
	}

	access, err := c.svc.Evaluate(r.Context(), tenantID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to evaluate entitlements")
	}
	grants, err := c.svc.Grants(r.Context(), tenantID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list entitlements")
	}

	items := make([]grantResponse, 0, len(grants))
	for _, grant := range grants {
		item := grantResponse{
			FeatureKey: grant.FeatureKey,
			Status:     grant.Status,
			Source:     grant.Source,
			StartsAt:   grant.StartsAt.UTC().Format(time.RFC3339),
		}
		if grant.EndsAt != nil {
			formatted := grant.EndsAt.UTC().Format(time.RFC3339)
			item.EndsAt = &formatted
		}
		items = append(items, item)
	}

	return api.Success(w, http.StatusOK, tenantEntitlementsResponse{
		TenantID: tenantID,
		Features: access.Features,
		Grants:   items,
	})
}

func (c *Controller) PutPlatformTenantEntitlements(w http.ResponseWriter, r *http.Request) error {
	tenantID := r.PathValue("id")
	if tenantID == "" {
		return appErrors.New(appErrors.CodeValidation, "tenant id is required")
	}
	if err := c.requireTenant(r, tenantID); err != nil {
		return err
	}

	var req setAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.New(appErrors.CodeValidation, "invalid request")
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	var endsAt *time.Time
	if req.EndsAt != nil && *req.EndsAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, "ends_at must be RFC3339")
		}
		utc := parsed.UTC()
		endsAt = &utc
	}

	err := c.svc.SetAccess(r.Context(), tenantID, application.SetAccessInput{
		Product: req.Product,
		Feature: req.Feature,
		Enabled: req.Enabled,
		EndsAt:  endsAt,
		ActorID: claims.UserID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUnknownFeature) || errors.Is(err, domain.ErrUnknownProduct) {
			return appErrors.Wrap(err, appErrors.CodeValidation, "unknown product or feature")
		}
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to update entitlements")
	}

	return c.GetPlatformTenantEntitlements(w, r)
}

func (c *Controller) requireTenant(r *http.Request, tenantID string) error {
	exists, err := c.tenants.TenantExists(r.Context(), tenantID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to lookup tenant")
	}
	if !exists {
		return appErrors.New(appErrors.CodeNotFound, "tenant not found")
	}
	return nil
}
