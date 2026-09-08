package v1

import (
	"fmt"
	"strings"

	"github.com/bowerbird/internal/tenant/domain"
)

type createTenantRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (r createTenantRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if strings.TrimSpace(r.Slug) == "" {
		return fmt.Errorf("slug is required")
	}

	if err := domain.ValidateSlug(r.Slug); err != nil {
		return err
	}

	return nil
}
