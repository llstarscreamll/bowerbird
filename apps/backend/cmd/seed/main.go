package main

import (
	"context"
	"log"
	"os"

	"github.com/bowerbird/internal/identity/domain"
	idinfra "github.com/bowerbird/internal/identity/infrastructure"
	organizationModule "github.com/bowerbird/internal/organization"
	"github.com/bowerbird/internal/organization/application"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
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

	migrationsDir := os.Getenv("TENANT_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations/tenant"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			migrationsDir = "apps/backend/migrations/tenant"
		}
	}

	organizationApp := organizationModule.NewApplication(pool, cfg.DatabaseURL, migrationsDir)
	orgUseCase := application.NewCreateOrganizationUseCaseFromCommand(organizationApp.Commands.CreateOrganization)

	// We also need the user to exist in the Control Plane identity tables before we create the tenant.
	// Because the AddMembership requires a foreign key to users.id
	dbRegistry := database.NewRegistry(pool, cfg.DatabaseURL)
	defer dbRegistry.CloseAll()
	idRepo := idinfra.NewPostgresRepository(pool, dbRegistry)

	email := "admin@acme.com"
	user, err := idRepo.FindUserByEmail(ctx, email)
	if err != nil && err != domain.ErrUserNotFound {
		log.Fatalf("Failed to lookup user: %v", err)
	}

	if user == nil {
		log.Println("Creating identity for admin@acme.com...")
		user = domain.NewUser("", email, "Admin", "Acme", "")
		if err := idRepo.CreateUser(ctx, user); err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		// Fetch again to get the auto-generated ID
		user, _ = idRepo.FindUserByEmail(ctx, email)
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
		if err == application.ErrSlugAlreadyExists {
			log.Println("Tenant 'acme' already exists, skipping.")
			return
		}

		log.Fatalf("Failed to create tenant: %v", err)
	}

	log.Printf("Successfully created tenant 'Acme Corp' with slug 'acme', DB name: %s", org.DBName)
}
