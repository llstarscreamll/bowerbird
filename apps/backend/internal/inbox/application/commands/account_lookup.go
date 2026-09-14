package commands

import (
	"context"
	"errors"
	"fmt"

	connectionsapi "github.com/bowerbird/internal/connections/api"
)

var ErrMailboxConnectionNotFound = errors.New("mailbox connection not found")

func decryptAccount(
	ctx context.Context,
	connectionsService connectionsapi.InternalService,
	accountID string,
) (connectionsapi.ConnectionInfo, []byte, error) {
	account, err := connectionsService.GetConnection(ctx, accountID)
	if err != nil {
		if errors.Is(err, connectionsapi.ErrConnectionNotFound) {
			return connectionsapi.ConnectionInfo{}, nil, ErrMailboxConnectionNotFound
		}
		return connectionsapi.ConnectionInfo{}, nil, fmt.Errorf("get account: %w", err)
	}

	credentialsJSON, err := connectionsService.DecryptCredentials(ctx, account.ID)
	if err != nil {
		return connectionsapi.ConnectionInfo{}, nil, fmt.Errorf("decrypt account credentials: %w", err)
	}
	return account, credentialsJSON, nil
}
