package main

import (
	"context"
	"log"

	"github.com/money-path/bowerbird/apps/backend/internal/identity/domain"
	idinfra "github.com/money-path/bowerbird/apps/backend/internal/identity/infrastructure"
	"github.com/money-path/bowerbird/apps/backend/internal/organization/application"
	"github.com/money-path/bowerbird/apps/backend/internal/organization/infrastructure"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/config"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/database"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	orgRepo := infrastructure.NewPostgresRepository(pool)
	orgProvisioner := infrastructure.NewPostgresProvisioner(pool, cfg.DatabaseURL, "migrations/tenant")
	orgUseCase := application.NewCreateOrganizationUseCase(orgRepo, orgProvisioner)

	// We also need the user to exist in the Control Plane identity tables before we create the tenant.
	// Because the AddMembership requires a foreign key to users.id
	dbRegistry := database.NewRegistry(pool, cfg.DatabaseURL)
	idRepo := idinfra.NewPostgresRepository(pool, dbRegistry)

	email := "admin@acme.com"
	user, err := idRepo.FindUserByEmail(ctx, email)
	if err != nil && err != domain.ErrUserNotFound {
		log.Fatalf("Failed to lookup user: %v", err)
	}

	if user == nil {
		log.Println("Creating identity for admin@acme.com...")
		user = domain.NewUser("", email)
		if err := idRepo.CreateUser(ctx, user); err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		// Fetch again to get the auto-generated ID
		user, _ = idRepo.FindUserByEmail(ctx, email)
	}

	// Check if already exists
	exists, err := orgRepo.ExistsBySlug(ctx, "acme")
	if err != nil {
		log.Fatalf("Failed to check if org exists: %v", err)
	}
	if exists {
		log.Println("Tenant 'acme' already exists, skipping.")
		return
	}

	cmd := application.CreateOrganizationCommand{
		Name:           "Acme Corp",
		Slug:           "acme",
		OwnerID:        user.ID,
		OwnerEmail:     user.Email,
		OwnerFirstName: "Admin",
		OwnerLastName:  "Acme",
		OwnerAvatarURL: "https://i.pravatar.cc/150?u=admin@acme.com",
	}

	org, err := orgUseCase.Execute(ctx, cmd)
	if err != nil {
		log.Fatalf("Failed to create tenant: %v", err)
	}

	log.Printf("Successfully created tenant 'Acme Corp' with slug 'acme', DB name: %s", org.DBName)
}
