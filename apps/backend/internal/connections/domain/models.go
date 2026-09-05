package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	ConnectionStatusActive            = "active"
	ConnectionStatusRequiresReconnect = "requires_reconnect"
	ConnectionStatusPaused            = "paused"
)

const (
	SharingPolicyPrivate   = "private"
	SharingPolicyTenantAll = "tenant_all"
)

var (
	ErrNilConnection                = errors.New("connection is nil")
	ErrInvalidSharingPolicy         = errors.New("invalid sharing policy")
	ErrConnectionIDRequired         = errors.New("connection id is required")
	ErrOwnerRequired                = errors.New("connection owner is required")
	ErrProviderRequired             = errors.New("connection provider is required")
	ErrProviderAccountEmailRequired = errors.New("provider account email is required")
	ErrCredentialsRequired          = errors.New("encrypted credentials are required")
	ErrReconnectReasonRequired      = errors.New("reconnect reason is required")
)

type Connection struct {
	ID                   string
	OwnerUserID          string
	Provider             string
	ProviderAccountEmail string
	Status               string
	EncryptedCredentials []byte
	GrantedScopes        []string
	SharingPolicy        string
	RawData              []byte
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewConnection(id, ownerUserID, provider, providerAccountEmail string, grantedScopes []string, sharingPolicy string, at time.Time) (*Connection, error) {
	id = strings.TrimSpace(id)
	ownerUserID = strings.TrimSpace(ownerUserID)
	provider = strings.TrimSpace(provider)
	providerAccountEmail = strings.TrimSpace(providerAccountEmail)
	if id == "" {
		return nil, ErrConnectionIDRequired
	}
	if ownerUserID == "" {
		return nil, ErrOwnerRequired
	}
	if provider == "" {
		return nil, ErrProviderRequired
	}
	if providerAccountEmail == "" {
		return nil, ErrProviderAccountEmailRequired
	}
	if sharingPolicy == "" {
		sharingPolicy = SharingPolicyPrivate
	}
	if sharingPolicy != SharingPolicyPrivate && sharingPolicy != SharingPolicyTenantAll {
		return nil, ErrInvalidSharingPolicy
	}

	at = at.UTC()
	scopes := append([]string(nil), grantedScopes...)
	return &Connection{
		ID:                   id,
		OwnerUserID:          ownerUserID,
		Provider:             provider,
		ProviderAccountEmail: providerAccountEmail,
		Status:               ConnectionStatusActive,
		GrantedScopes:        scopes,
		SharingPolicy:        sharingPolicy,
		CreatedAt:            at,
		UpdatedAt:            at,
	}, nil
}

func (c *Connection) SealCredentials(ciphertext []byte, at time.Time) error {
	if c == nil {
		return ErrNilConnection
	}
	if len(ciphertext) == 0 {
		return ErrCredentialsRequired
	}
	c.EncryptedCredentials = append([]byte(nil), ciphertext...)
	c.UpdatedAt = at.UTC()
	return nil
}

func (c *Connection) MarkRequiresReconnect(reason string, at time.Time) error {
	if c == nil {
		return ErrNilConnection
	}
	if strings.TrimSpace(reason) == "" {
		return ErrReconnectReasonRequired
	}
	c.Status = ConnectionStatusRequiresReconnect
	c.UpdatedAt = at.UTC()
	return nil
}

func (c *Connection) MarkActive(at time.Time) error {
	if c == nil {
		return ErrNilConnection
	}
	c.Status = ConnectionStatusActive
	c.UpdatedAt = at.UTC()
	return nil
}

func (c *Connection) UpdateSharingPolicy(policy string, at time.Time) error {
	if c == nil {
		return ErrNilConnection
	}
	if policy != SharingPolicyPrivate && policy != SharingPolicyTenantAll {
		return ErrInvalidSharingPolicy
	}
	c.SharingPolicy = policy
	c.UpdatedAt = at.UTC()
	return nil
}

type Repository interface {
	GetByID(ctx context.Context, id string) (*Connection, error)
	ListAll(ctx context.Context) ([]*Connection, error)
	ListActive(ctx context.Context) ([]*Connection, error)
	ListByOwner(ctx context.Context, ownerUserID string) ([]*Connection, error)
	Upsert(ctx context.Context, conn *Connection) error
	Delete(ctx context.Context, id string) error
}
