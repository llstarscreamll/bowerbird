package v1

import (
	"fmt"
	"strings"
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

	return nil
}
