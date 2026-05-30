package application

import (
	"context"
	"fmt"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/connections/domain"
)

type internalService struct {
	repo               domain.Repository
	credentialsService *CredentialsService
}

func NewInternalService(
	repo domain.Repository,
	credentialsService *CredentialsService,
) InternalService {
	return &internalService{
		repo:               repo,
		credentialsService: credentialsService,
	}
}

func (s *internalService) GetActiveConnections(ctx context.Context) ([]ConnectionInfo, error) {
	connections, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active connections: %w", err)
	}

	result := make([]ConnectionInfo, 0, len(connections))
	for _, c := range connections {
		result = append(result, ConnectionInfo{
			ID:                   c.ID,
			Provider:             c.Provider,
			ProviderAccountEmail: c.ProviderAccountEmail,
			OwnerUserID:          c.OwnerUserID,
			SharingPolicy:        c.SharingPolicy,
		})
	}
	return result, nil
}

func (s *internalService) DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error) {
	conn, err := s.repo.GetByID(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("get connection %s: %w", connectionID, err)
	}
	if conn == nil {
		return nil, fmt.Errorf("connection not found")
	}

	return s.credentialsService.ReadDecryptedCredentials(conn)
}

func (s *internalService) MarkRequiresReconnect(ctx context.Context, connectionID, reason string) error {
	conn, err := s.repo.GetByID(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("get connection %s: %w", connectionID, err)
	}
	if conn == nil {
		return fmt.Errorf("connection not found")
	}

	// For simplicity, using time.Now(). In a full setup we inject time generator.
	// But let's just use domain model.
	// wait, time is not injected, let's just use standard time or add it if needed.
	// We'll leave it as we just update repo directly or use model method
	if err := conn.MarkRequiresReconnect(reason, time.Now()); err != nil {
		return err
	}

	return s.repo.Upsert(ctx, conn)
}

func (s *internalService) GetSharingPolicy(ctx context.Context, connectionID string) (string, error) {
	conn, err := s.repo.GetByID(ctx, connectionID)
	if err != nil {
		return "", fmt.Errorf("get connection %s: %w", connectionID, err)
	}
	if conn == nil {
		return "", fmt.Errorf("connection not found")
	}

	return conn.SharingPolicy, nil
}
