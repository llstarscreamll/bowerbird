package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
	"github.com/bowerbird/internal/platform/id"
)

type CreateInvoiceInput struct {
	SourceName       string
	SourceID         string
	ExtractionSource string
	StorageKey       string
	Invoice          *domain.InvoiceDocument
}

type CreateInvoiceResult struct {
	HeaderID string
	LineIDs  []string
}

type CreateInvoiceCommand struct {
	repo          ports.InvoiceWriteRepository
	partyResolver ports.IssuerPartyResolver
	lineResolver  ports.CatalogLineResolver
	logger        *slog.Logger
	now           func() time.Time
	newID         func() string
}

func NewCreateInvoiceCommand(
	repo ports.InvoiceWriteRepository,
	partyResolver ports.IssuerPartyResolver,
	lineResolver ports.CatalogLineResolver,
) *CreateInvoiceCommand {
	if repo == nil {
		panic("invoice write repository is required")
	}
	if partyResolver == nil {
		panic("issuer party resolver is required")
	}
	if lineResolver == nil {
		panic("catalog line resolver is required")
	}

	return &CreateInvoiceCommand{
		repo:          repo,
		partyResolver: partyResolver,
		lineResolver:  lineResolver,
		logger:        slog.Default(),
		now:           time.Now,
		newID:         id.NewULID,
	}
}

func (cmd *CreateInvoiceCommand) Execute(ctx context.Context, input CreateInvoiceInput) (*CreateInvoiceResult, error) {
	if input.Invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}
	if err := input.Invoice.Validate(); err != nil {
		return nil, err
	}

	now := cmd.now().UTC()
	headerID := cmd.newID()
	headerRawData, err := normalizeInvoiceRawData(input.Invoice.RawData)
	if err != nil {
		return nil, fmt.Errorf("normalize invoice raw data: %w", err)
	}

	header := input.Invoice.ToHeaderRecord(domain.InvoiceReceipt{
		ID:               headerID,
		SourceName:       input.SourceName,
		SourceID:         input.SourceID,
		ExtractionSource: input.ExtractionSource,
		StorageKey:       input.StorageKey,
		RawData:          headerRawData,
		ReceivedAt:       now,
	})

	lines := make([]domain.InvoiceLineRecord, 0, len(input.Invoice.Lines))
	lineIDs := make([]string, 0, len(input.Invoice.Lines))
	for idx, line := range input.Invoice.Lines {
		lineID := cmd.newID()
		lineRawData, err := json.Marshal(line)
		if err != nil {
			return nil, fmt.Errorf("marshal invoice line raw data: %w", err)
		}

		lines = append(lines, line.ToLineRecord(lineID, headerID, idx+1, lineRawData, now))
		lineIDs = append(lineIDs, lineID)
	}

	if err := cmd.repo.PersistInvoiceAtomic(ctx, header, lines); err != nil {
		return nil, err
	}
	cmd.logger.Info("invoice persisted atomically", "header_id", headerID, "cufe", header.CUFE, "lines", len(lines))

	if err := cmd.applyLinking(ctx, header, lines); err != nil {
		cmd.logger.Error("invoice catalog linking failed after persist", "header_id", headerID, "error", err)
		return &CreateInvoiceResult{HeaderID: headerID, LineIDs: lineIDs}, fmt.Errorf("catalog linking: %w", err)
	}

	return &CreateInvoiceResult{HeaderID: headerID, LineIDs: lineIDs}, nil
}

func (cmd *CreateInvoiceCommand) applyLinking(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	var partyID string
	var resolveErr error
	resolved, err := cmd.partyResolver.ResolveIssuerPartyID(ctx, header.IssuerTaxID, header.IssuerName)
	if err != nil {
		resolveErr = err
	} else {
		partyID = resolved
	}

	updates := make([]ports.LineLinkUpdate, 0, len(lines))
	if resolveErr == nil {
		for _, line := range lines {
			result, err := cmd.lineResolver.ResolveLine(ctx, ports.CatalogLineResolveInput{
				LineID:      line.ID,
				PartyID:     partyID,
				ItemCode:    line.ItemCode,
				Description: line.Description,
			})
			if err != nil {
				resolveErr = err
				break
			}
			link := domain.LineLinkFromRecord(
				result.ItemID,
				result.Status,
				result.Method,
				false,
				normalizeSuggestionsJSON(result.Suggestions),
			)
			updates = append(updates, ports.NewLineLinkUpdate(line.ID, link))
		}
	}

	var partyPtr *string
	if partyID != "" {
		partyPtr = &partyID
	}
	statuses := make([]string, 0, len(updates))
	for _, u := range updates {
		statuses = append(statuses, u.LinkStatus)
	}
	linkingStatus := domain.RecalculateLinkingStatus(statuses)
	if resolveErr != nil {
		linkingStatus = domain.LinkingStatusFailed
	}

	if err := cmd.repo.ApplyCatalogLinking(ctx, header.ID, partyPtr, linkingStatus, updates); err != nil {
		cmd.logger.Error("failed to persist invoice catalog linking state", "header_id", header.ID, "error", err)
		if resolveErr != nil {
			return fmt.Errorf("resolve linking: %w; also failed to persist linking state: %v", resolveErr, err)
		}
		return fmt.Errorf("persist linking state: %w", err)
	}
	return resolveErr
}

func normalizeSuggestionsJSON(raw []byte) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return []byte("[]")
	}
	return raw
}

func normalizeInvoiceRawData(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("{}"), nil
	}
	if json.Valid(raw) {
		return raw, nil
	}

	normalized, err := json.Marshal(string(raw))
	if err != nil {
		return nil, err
	}

	return normalized, nil
}
