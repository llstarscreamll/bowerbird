package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
)

type RecordMatchMemoryCommand struct {
	memories ports.MatchMemoryRepository
	now      func() time.Time
	newID    func() string
}

func NewRecordMatchMemoryCommand(memories ports.MatchMemoryRepository) *RecordMatchMemoryCommand {
	return &RecordMatchMemoryCommand{
		memories: memories,
		now:      time.Now,
		newID:    id.NewULID,
	}
}

type RecordMatchMemoryInput struct {
	PartyID     string
	ItemCode    string
	Description string
	Action      string
	ItemID      *string
}

func (cmd *RecordMatchMemoryCommand) Execute(ctx context.Context, input RecordMatchMemoryInput) error {
	if cmd.memories == nil {
		return fmt.Errorf("match memory repository is required")
	}
	mem, err := domain.NewMatchMemory(cmd.newID(), input.PartyID, input.ItemCode, input.Description, input.Action, input.ItemID, cmd.now())
	if err != nil {
		if errors.Is(err, domain.ErrInvalidMemoryAction) {
			return appErrors.New(appErrors.CodeValidation, "invalid memory action")
		}
		if errors.Is(err, domain.ErrItemIDRequired) {
			return appErrors.New(appErrors.CodeValidation, "item_id is required for link")
		}
		return err
	}
	return cmd.memories.UpsertMemory(ctx, mem)
}
