package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bowerbird/internal/connections/application/ports"
	"github.com/bowerbird/internal/connections/domain"
	"github.com/bowerbird/internal/platform/id"
)

type UpsertMailboxConnectionInput struct {
	ID                   string
	OwnerUserID          string
	Provider             string
	ProviderAccountEmail string
	GrantedScopes        []string
	SharingPolicy        string
	TokenJSON            []byte
}

type UpsertMailboxConnectionCommand struct {
	repo        ports.ConnectionRepository
	credentials ports.Credentials
	now         func() time.Time
	newID       func() string
}

func NewUpsertMailboxConnectionCommand(repo ports.ConnectionRepository, credentials ports.Credentials) *UpsertMailboxConnectionCommand {
	if repo == nil {
		panic("connection repository is required")
	}
	if credentials == nil {
		panic("credentials are required")
	}
	return &UpsertMailboxConnectionCommand{
		repo:        repo,
		credentials: credentials,
		now:         time.Now,
		newID:       id.NewULID,
	}
}

func (cmd *UpsertMailboxConnectionCommand) Execute(ctx context.Context, input UpsertMailboxConnectionInput) (*domain.Connection, error) {
	now := cmd.now().UTC()
	connectionID := input.ID
	if connectionID == "" {
		connectionID = cmd.newID()
	}

	conn, err := domain.NewConnection(
		connectionID,
		input.OwnerUserID,
		input.Provider,
		input.ProviderAccountEmail,
		input.GrantedScopes,
		input.SharingPolicy,
		now,
	)
	if err != nil {
		return nil, err
	}

	ciphertext, err := cmd.credentials.Encrypt(input.TokenJSON)
	if err != nil {
		return nil, err
	}
	if err := conn.SealCredentials(ciphertext, now); err != nil {
		return nil, err
	}
	if err := cmd.repo.Upsert(ctx, conn); err != nil {
		return nil, fmt.Errorf("save connection: %w", err)
	}
	return conn, nil
}
