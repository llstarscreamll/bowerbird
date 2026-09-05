package commands

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type ApplyLineDecisionCommand struct {
	links   ports.InvoiceLineLinkRepository
	catalog ports.CatalogMatchingPort
	logger  *slog.Logger
}

func NewApplyLineDecisionCommand(links ports.InvoiceLineLinkRepository, catalog ports.CatalogMatchingPort) *ApplyLineDecisionCommand {
	if links == nil {
		panic("invoice line link repository is required")
	}
	if catalog == nil {
		panic("catalog matching port is required")
	}

	return &ApplyLineDecisionCommand{
		links:   links,
		catalog: catalog,
		logger:  slog.Default(),
	}
}

type ApplyLineDecisionInput struct {
	InvoiceID   string
	LineID      string
	ItemID      string
	Action      string // link | never_match | create_provisional
	Remember    bool
	Lock        bool
	PartyID     string
	ItemCode    string
	Description string
}

func (cmd *ApplyLineDecisionCommand) Execute(ctx context.Context, input ApplyLineDecisionInput) error {
	line, err := cmd.links.GetLineForDecision(ctx, input.LineID)
	if err != nil {
		return err
	}
	if line == nil {
		return appErrors.New(appErrors.CodeNotFound, "invoice line not found")
	}
	if !line.BelongsToInvoice(input.InvoiceID) {
		return appErrors.New(appErrors.CodeNotFound, "invoice line not found")
	}

	partyID := firstNonEmpty(input.PartyID, line.PartyID)
	itemCode := firstNonEmpty(input.ItemCode, line.ItemCode)
	description := firstNonEmpty(input.Description, line.Description)

	action := strings.TrimSpace(input.Action)
	if action == "" {
		action = domain.MemoryActionLink
	}
	if action == domain.ActionCreateProvisional {
		itemID, err := cmd.catalog.MintProvisionalFromEvidence(ctx, ports.MintProvisionalInput{
			PartyID:     partyID,
			ItemCode:    itemCode,
			Description: description,
		})
		if err != nil {
			cmd.logDecision(input.LineID, line.InvoiceHeaderID, action, "catalog_mint_failed")
			return err
		}
		input.ItemID = itemID
		action = domain.MemoryActionLink
		input.Remember = true
		input.Lock = true
	}

	next, err := line.Link.ApplyManualDecision(action, input.ItemID, input.Lock)
	if err != nil {
		if errors.Is(err, domain.ErrLineLinkLocked) {
			return appErrors.New(appErrors.CodeConflict, "line link is locked")
		}
		if errors.Is(err, domain.ErrItemIDRequired) {
			return appErrors.New(appErrors.CodeValidation, "item_id is required for link")
		}
		if errors.Is(err, domain.ErrInvalidAction) {
			return appErrors.New(appErrors.CodeValidation, "invalid memory action")
		}
		return err
	}

	if next.IsLinked() {
		if err := cmd.catalog.ValidateItemExists(ctx, input.ItemID); err != nil {
			cmd.logDecision(input.LineID, line.InvoiceHeaderID, action, "validate_item_failed")
			return err
		}
		code := strings.TrimSpace(itemCode)
		if input.Remember && code != "" && partyID != "" {
			if err := cmd.catalog.EnsureSupplierAlias(ctx, partyID, code, input.ItemID); err != nil {
				cmd.logDecision(input.LineID, line.InvoiceHeaderID, action, "ensure_alias_failed")
				return err
			}
		}
	}

	// ACL before persist: catalog failure must not mutate the invoice line.
	if input.Remember {
		memItemID := domain.RememberedItemID(action, next.ItemID, input.ItemID)
		if err := cmd.catalog.RecordMatchMemory(ctx, ports.MatchMemoryInput{
			PartyID:     partyID,
			ItemCode:    itemCode,
			Description: description,
			Action:      action,
			ItemID:      memItemID,
		}); err != nil {
			cmd.logDecision(input.LineID, line.InvoiceHeaderID, action, "record_memory_failed")
			return err
		}
	}

	if err := cmd.links.SaveLineLink(ctx, input.LineID, next); err != nil {
		cmd.logDecision(input.LineID, line.InvoiceHeaderID, action, "save_failed")
		return err
	}

	if line.InvoiceHeaderID != "" {
		if err := cmd.links.SyncHeaderLinkingStatus(ctx, line.InvoiceHeaderID); err != nil {
			return err
		}
	}

	cmd.logDecision(input.LineID, line.InvoiceHeaderID, action, "ok")
	return nil
}

func (cmd *ApplyLineDecisionCommand) logDecision(lineID, headerID, action, outcome string) {
	cmd.logger.Info("invoices.apply_line_decision",
		"line_id", lineID,
		"invoice_header_id", headerID,
		"action", action,
		"outcome", outcome,
	)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
