package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidSlug = errors.New("invalid tenant slug: must be alphanumeric and hyphens only")
)

const (
	StatusProvisioning = "provisioning"
	StatusActive       = "active"
	StatusFailed       = "failed"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

// Tenant represents a tenant in the system.
type Tenant struct {
	ID              string
	Name            string
	Slug            string
	DBName          string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	MembersCount    int
	CurrentUserRole string
}

// NewTenant creates a new tenant entity with valid defaults.
func NewTenant(name, slug string) (*Tenant, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))

	if !slugRegex.MatchString(slug) {
		return nil, ErrInvalidSlug
	}

	dbName := "tenant_" + strings.ReplaceAll(slug, "-", "_")

	return &Tenant{
		Name:      strings.TrimSpace(name),
		Slug:      slug,
		DBName:    dbName,
		Status:    StatusProvisioning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}
