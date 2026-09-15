package commands

import (
	"context"
)

type HydrateMessageCommand struct {
	sync *SyncAccountCommand
}

func NewHydrateMessageCommand(sync *SyncAccountCommand) *HydrateMessageCommand {
	if sync == nil {
		return nil
	}
	return &HydrateMessageCommand{sync: sync}
}

func (c *HydrateMessageCommand) Execute(ctx context.Context, messageID string) error {
	if c == nil || c.sync == nil {
		return nil
	}
	return c.sync.hydrateMessage(ctx, messageID)
}
