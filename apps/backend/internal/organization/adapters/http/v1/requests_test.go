package v1

import "testing"

func TestCreateOrganizationRequestValidateSuccess(t *testing.T) {
	req := createOrganizationRequest{
		Name: "Acme Corp",
		Slug: "acme-corp",
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}

func TestCreateOrganizationRequestValidateMissingName(t *testing.T) {
	req := createOrganizationRequest{Slug: "acme-corp"}

	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateOrganizationRequestValidateMissingSlug(t *testing.T) {
	req := createOrganizationRequest{Name: "Acme Corp"}

	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
