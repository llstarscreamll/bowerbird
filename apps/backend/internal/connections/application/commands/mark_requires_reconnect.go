package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bowerbird/internal/connections/application/ports"
)

type MarkRequiresReconnectCommand struct {
	repo ports.ConnectionRepository
	now  func() time.Time
}

func NewMarkRequiresReconnectCommand(repo ports.ConnectionRepository) *MarkRequiresReconnectCommand {
	if repo == nil {
		panic("connection repository is required")
	}

	return &MarkRequiresReconnectCommand{repo: repo, now: time.Now}
}

func (cmd *MarkRequiresReconnectCommand) Execute(ctx context.Context, connectionID, reason string) error {
	conn, err := cmd.repo.GetByID(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("get connection %s: %w", connectionID, err)
	}
	if conn == nil {
		return fmt.Errorf("connection not found")
	}

	if err := conn.MarkRequiresReconnect(reason, cmd.now()); err != nil {
		return err
	}

	return cmd.repo.Upsert(ctx, conn)
}
