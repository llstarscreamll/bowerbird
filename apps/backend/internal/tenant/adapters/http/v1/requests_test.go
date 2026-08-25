package v1

import "testing"

func TestCreateTenantRequestValidateSuccess(t *testing.T) {
	req := createTenantRequest{
		Name: "Acme Corp",
		Slug: "acme-corp",
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}

func TestCreateTenantRequestValidateMissingName(t *testing.T) {
	req := createTenantRequest{Slug: "acme-corp"}

	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateTenantRequestValidateMissingSlug(t *testing.T) {
	req := createTenantRequest{Name: "Acme Corp"}

	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
