package commands

import (
	"context"
	"fmt"

	connectionsapi "github.com/bowerbird/internal/connections/api"
)

func decryptActiveAccount(
	ctx context.Context,
	connectionsService connectionsapi.InternalService,
	accountID string,
) (connectionsapi.ConnectionInfo, []byte, error) {
	accounts, err := connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return connectionsapi.ConnectionInfo{}, nil, fmt.Errorf("list active accounts: %w", err)
	}

	for _, account := range accounts {
		if account.ID == accountID {
			credentialsJSON, err := connectionsService.DecryptCredentials(ctx, account.ID)
			if err != nil {
				return connectionsapi.ConnectionInfo{}, nil, fmt.Errorf("decrypt account credentials: %w", err)
			}
			return account, credentialsJSON, nil
		}
	}

	return connectionsapi.ConnectionInfo{}, nil, fmt.Errorf("active account not found: %s", accountID)
}
