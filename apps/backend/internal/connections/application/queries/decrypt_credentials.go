package queries

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/connections/application/commands"
	"github.com/bowerbird/internal/connections/application/ports"
)

type DecryptCredentialsQuery struct {
	repo               ports.ConnectionRepository
	credentialsService *commands.CredentialsService
}

func NewDecryptCredentialsQuery(repo ports.ConnectionRepository, credentialsService *commands.CredentialsService) *DecryptCredentialsQuery {
	if repo == nil {
		panic("connection repository is required")
	}

	return &DecryptCredentialsQuery{
		repo:               repo,
		credentialsService: credentialsService,
	}
}

func (q *DecryptCredentialsQuery) Execute(ctx context.Context, connectionID string) ([]byte, error) {
	conn, err := q.repo.GetByID(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("get connection %s: %w", connectionID, err)
	}
	if conn == nil {
		return nil, fmt.Errorf("connection not found")
	}

	if q.credentialsService == nil {
		return nil, commands.ErrCipherNotConfigured
	}

	return q.credentialsService.ReadDecryptedCredentials(conn)
}
