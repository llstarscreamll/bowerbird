package commands

import (
	"context"
	"fmt"

	connectionsApp "github.com/bowerbird/internal/connections/application"
)

func decryptActiveAccount(
	ctx context.Context,
	connectionsService connectionsApp.InternalService,
	accountID string,
) (connectionsApp.ConnectionInfo, []byte, error) {
	accounts, err := connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return connectionsApp.ConnectionInfo{}, nil, fmt.Errorf("list active accounts: %w", err)
	}

	for _, account := range accounts {
		if account.ID == accountID {
			credentialsJSON, err := connectionsService.DecryptCredentials(ctx, account.ID)
			if err != nil {
				return connectionsApp.ConnectionInfo{}, nil, fmt.Errorf("decrypt account credentials: %w", err)
			}
			return account, credentialsJSON, nil
		}
	}

	return connectionsApp.ConnectionInfo{}, nil, fmt.Errorf("active account not found: %s", accountID)
}
