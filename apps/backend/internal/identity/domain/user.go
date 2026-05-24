package domain

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrIdentityAlreadyExists = errors.New("user identity already exists")
)

// User represents the identity in the Control Plane
type User struct {
	ID         string
	Email      string
	FirstName  string
	LastName   string
	PictureURL string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

// UserIdentity represents a linked authentication provider for a user
type UserIdentity struct {
	ID         string
	UserID     string
	Provider   string
	ProviderID string
	CreatedAt  time.Time
}

// TenantMembershipRole defines the role of a user in a tenant
type TenantMembershipRole string

const (
	RoleOwner  TenantMembershipRole = "OWNER"
	RoleAdmin  TenantMembershipRole = "ADMIN"
	RoleMember TenantMembershipRole = "MEMBER"
)

// TenantMembership represents a user's association with a tenant
type TenantMembership struct {
	UserID    string
	TenantID  string
	Role      TenantMembershipRole
	CreatedAt time.Time
	DeletedAt *time.Time
}

// TenantUserProfile represents the user's data stored in the Tenant Database
type TenantUserProfile struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	AvatarURL string
	Status    string // active, inactive
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// NewUser creates a new User
func NewUser(id, email, firstName, lastName, pictureURL string) *User {
	now := time.Now()
	return &User{
		ID:         id,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		PictureURL: pictureURL,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewUserIdentity creates a new UserIdentity
func NewUserIdentity(id, userID, provider, providerID string) *UserIdentity {
	return &UserIdentity{
		ID:         id,
		UserID:     userID,
		Provider:   provider,
		ProviderID: providerID,
		CreatedAt:  time.Now(),
	}
}

// NewTenantMembership creates a new TenantMembership
func NewTenantMembership(userID, tenantID string, role TenantMembershipRole) *TenantMembership {
	return &TenantMembership{
		UserID:    userID,
		TenantID:  tenantID,
		Role:      role,
		CreatedAt: time.Now(),
	}
}
