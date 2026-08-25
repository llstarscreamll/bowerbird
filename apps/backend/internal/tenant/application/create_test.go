package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bowerbird/internal/tenant/domain"
)

type fakeTenantRepo struct {
	existsBySlug     bool
	createErr        error
	addMembershipErr error
	updateStatusErr  error

	createdOrg     *domain.Tenant
	createdStatus  string
	statusUpdates  []string
	membershipAdds int
}

func (r *fakeTenantRepo) Create(ctx context.Context, org *domain.Tenant) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.createdOrg = org
	r.createdStatus = org.Status
	return nil
}

func (r *fakeTenantRepo) UpdateStatus(ctx context.Context, tenantID, status string) error {
	if r.updateStatusErr != nil {
		return r.updateStatusErr
	}
	r.statusUpdates = append(r.statusUpdates, status)
	return nil
}

func (r *fakeTenantRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return r.existsBySlug, nil
}

func (r *fakeTenantRepo) GetByID(ctx context.Context, id, userID string) (*domain.Tenant, error) {
	return nil, nil // Dummy implementation for test
}

func (r *fakeTenantRepo) AddMembership(ctx context.Context, userID, tenantID, role string) error {
	if r.addMembershipErr != nil {
		return r.addMembershipErr
	}
	r.membershipAdds++
	return nil
}

func (r *fakeTenantRepo) ListAll(ctx context.Context) ([]domain.Tenant, error) {
	return nil, nil
}

func (r *fakeTenantRepo) ExistsByID(ctx context.Context, id string) (bool, error) {
	return false, nil
}

type fakeProvisioner struct {
	createDatabaseErr  error
	migrateDatabaseErr error
	seedOwnerErr       error
}

func (p *fakeProvisioner) CreateDatabase(ctx context.Context, dbName string) error {
	return p.createDatabaseErr
}

func (p *fakeProvisioner) MigrateDatabase(ctx context.Context, dbName string) error {
	return p.migrateDatabaseErr
}

func (p *fakeProvisioner) SeedOwner(ctx context.Context, dbName string, owner domain.OwnerData) error {
	return p.seedOwnerErr
}

func TestCreateTenantStartsProvisioningAndEndsActive(t *testing.T) {
	repo := &fakeTenantRepo{}
	provisioner := &fakeProvisioner{}
	uc := NewCreateTenantUseCase(repo, provisioner)

	org, err := uc.Execute(context.Background(), CreateTenantCommand{
		Name:           "Acme",
		Slug:           "acme",
		OwnerID:        "user-1",
		OwnerEmail:     "owner@acme.com",
		OwnerFirstName: "Owner",
		OwnerLastName:  "One",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.createdOrg == nil {
		t.Fatal("expected tenant to be created")
	}
	if repo.createdStatus != domain.StatusProvisioning {
		t.Fatalf("expected created tenant status provisioning, got %s", repo.createdStatus)
	}
	if org.Status != domain.StatusActive {
		t.Fatalf("expected returned tenant status active, got %s", org.Status)
	}
	if len(repo.statusUpdates) != 1 || repo.statusUpdates[0] != domain.StatusActive {
		t.Fatalf("expected one status update to active, got %#v", repo.statusUpdates)
	}
}

func TestCreateTenantMarksFailedWhenProvisioningFails(t *testing.T) {
	repo := &fakeTenantRepo{}
	provisioner := &fakeProvisioner{createDatabaseErr: errors.New("db down")}
	uc := NewCreateTenantUseCase(repo, provisioner)

	_, err := uc.Execute(context.Background(), CreateTenantCommand{
		Name:           "Acme",
		Slug:           "acme",
		OwnerID:        "user-1",
		OwnerEmail:     "owner@acme.com",
		OwnerFirstName: "Owner",
		OwnerLastName:  "One",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to provision database") {
		t.Fatalf("expected provisioning error, got %v", err)
	}
	if len(repo.statusUpdates) != 1 || repo.statusUpdates[0] != domain.StatusFailed {
		t.Fatalf("expected one status update to failed, got %#v", repo.statusUpdates)
	}
}
