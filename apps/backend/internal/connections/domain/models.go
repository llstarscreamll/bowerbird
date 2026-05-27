package domain

import (
	"context"
	"errors"
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
	ErrNilConnection        = errors.New("connection is nil")
	ErrInvalidSharingPolicy = errors.New("invalid sharing policy")
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

func (c *Connection) MarkRequiresReconnect(reason string, at time.Time) error {
	if c == nil {
		return ErrNilConnection
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
