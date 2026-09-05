package queries

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/connections/application/ports"
)

type DecryptCredentialsQuery struct {
	repo        ports.ConnectionRepository
	credentials ports.Credentials
}

func NewDecryptCredentialsQuery(repo ports.ConnectionRepository, credentials ports.Credentials) *DecryptCredentialsQuery {
	if repo == nil {
		panic("connection repository is required")
	}
	if credentials == nil {
		panic("credentials are required")
	}

	return &DecryptCredentialsQuery{
		repo:        repo,
		credentials: credentials,
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

	return q.credentials.Decrypt(conn.EncryptedCredentials)
}
