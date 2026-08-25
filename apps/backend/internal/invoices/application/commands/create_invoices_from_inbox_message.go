package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	contractEvents "github.com/bowerbird/internal/contracts/events"
	"github.com/bowerbird/internal/invoices/application/ports"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/jobs"
)

type CreateInvoicesFromInboxMessageCommand struct {
	jobQueue jobs.Queue
	logger   *slog.Logger
	now      func() time.Time
	newID    func() string
}

func NewCreateInvoicesFromInboxMessageCommand(jobQueue jobs.Queue) *CreateInvoicesFromInboxMessageCommand {
	return &CreateInvoicesFromInboxMessageCommand{
		jobQueue: jobQueue,
		logger:   slog.Default(),
		now:      time.Now,
		newID:    id.NewULID,
	}
}

func (cmd *CreateInvoicesFromInboxMessageCommand) Execute(ctx context.Context, event contractEvents.InboxMessageReceived) error {
	if !hasInvoiceKeyword(event.Subject, event.Body) {
		cmd.logger.Info("invoicing event skipped: missing invoice keyword", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID)
		return nil
	}

	if !hasSupportedAttachment(event.AttachmentRefs) {
		cmd.logger.Info("invoicing event skipped: missing supported attachments", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID)
		return nil
	}

	if cmd.jobQueue == nil {
		cmd.logger.Info("invoicing candidate detected but queue not configured", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID)
		return nil
	}

	job := contractJobs.ExtractInvoicesFromFilesJob{
		ID:         cmd.newID(),
		SourceName: "inbox-message",
		SourceID:   event.MessageInternalID,
		Files:      mapAttachmentRefs(event.AttachmentRefs),
		QueuedAt:   cmd.now().UTC().Format(time.RFC3339Nano),
	}

	payload, err := contractJobs.MarshalInvoiceExtractionRequested(job)
	if err != nil {
		return err
	}

	err = cmd.jobQueue.Dispatch(ctx, jobs.Job{
		Type:    contractJobs.InvoiceExtractionRequestedType,
		Payload: payload,
	})
	if err != nil {
		return err
	}

	cmd.logger.Info("invoice extraction job queued", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID, "attachments", len(event.AttachmentRefs))
	return nil
}

func mapAttachmentRefs(refs []contractEvents.AttachmentRef) []contractJobs.File {
	mapped := make([]contractJobs.File, 0, len(refs))
	for _, ref := range refs {
		mapped = append(mapped, contractJobs.File{
			Path:     ref.S3Key,
			Filename: ref.Filename,
			MimeType: ref.MimeType,
		})
	}

	return mapped
}

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
	if cmd.repo == nil {
		return nil, fmt.Errorf("invoice write repository is required")
	}
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

	header := domain.InvoiceHeaderRecord{
		ID:               headerID,
		SourceName:       input.SourceName,
		SourceID:         input.SourceID,
		CUFE:             input.Invoice.CUFE,
		InvoiceNumber:    input.Invoice.InvoiceID,
		IssuerName:       input.Invoice.Issuer.Name,
		IssuerTaxID:      input.Invoice.Issuer.TaxID,
		ReceiverName:     input.Invoice.Receiver.Name,
		ReceiverTaxID:    input.Invoice.Receiver.TaxID,
		CurrencyCode:     input.Invoice.CurrencyCode,
		IssueDate:        input.Invoice.IssueDateTimeUTC(),
		PaymentCode:      input.Invoice.PaymentMeansCode,
		Subtotal:         input.Invoice.LineExtension,
		TaxTotal:         input.Invoice.TaxAmountTotal(),
		GrandTotal:       input.Invoice.PayableAmount,
		DocumentRefS3Key: input.StorageKey,
		ExtractionSource: input.ExtractionSource,
		RawData:          headerRawData,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	lines := make([]domain.InvoiceLineRecord, 0, len(input.Invoice.Lines))
	lineIDs := make([]string, 0, len(input.Invoice.Lines))
	for idx, line := range input.Invoice.Lines {
		lineID := cmd.newID()
		lineNumber := line.NumberOrDefault(idx + 1)
		lineRawData, err := json.Marshal(line)
		if err != nil {
			return nil, fmt.Errorf("marshal invoice line raw data: %w", err)
		}

		lines = append(lines, domain.InvoiceLineRecord{
			ID:              lineID,
			InvoiceHeaderID: headerID,
			LineNumber:      lineNumber,
			ItemCode:        line.ItemCode,
			Description:     line.ItemDescription,
			Quantity:        line.Quantity,
			UnitPrice:       line.UnitPrice,
			LineTaxTotal:    line.TaxAmount,
			LineTotal:       line.LineExtension,
			RawData:         lineRawData,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
		lineIDs = append(lineIDs, lineID)
	}

	if err := cmd.repo.PersistInvoiceAtomic(ctx, header, lines); err != nil {
		return nil, err
	}
	cmd.logger.Info("invoice persisted atomically", "header_id", headerID, "cufe", header.CUFE, "lines", len(lines))

	if err := cmd.applyLinking(ctx, header, lines); err != nil {
		cmd.logger.Error("invoice catalog linking failed after persist", "header_id", headerID, "error", err)
		// Invoice financial write is kept; applyLinking persists partial results + failed/pending status when possible.
	}

	return &CreateInvoiceResult{HeaderID: headerID, LineIDs: lineIDs}, nil
}

func (cmd *CreateInvoiceCommand) applyLinking(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	if cmd.partyResolver == nil && cmd.lineResolver == nil {
		return nil
	}

	var partyID string
	var resolveErr error
	if cmd.partyResolver != nil {
		resolved, err := cmd.partyResolver.ResolveIssuerPartyID(ctx, header.IssuerTaxID, header.IssuerName)
		if err != nil {
			resolveErr = err
		} else {
			partyID = resolved
		}
	}

	updates := make([]ports.LineLinkUpdate, 0, len(lines))
	if resolveErr == nil && cmd.lineResolver != nil {
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
			var itemID *string
			if result.ItemID != "" {
				idCopy := result.ItemID
				itemID = &idCopy
			}
			suggestions := normalizeSuggestionsJSON(result.Suggestions)
			updates = append(updates, ports.LineLinkUpdate{
				LineID:      line.ID,
				ItemID:      itemID,
				LinkStatus:  result.Status,
				LinkMethod:  result.Method,
				Suggestions: suggestions,
			})
		}
	}

	var partyPtr *string
	if partyID != "" {
		partyPtr = &partyID
	}
	linkingStatus := "linked"
	if resolveErr != nil {
		linkingStatus = "failed"
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

func hasSupportedAttachment(refs []contractEvents.AttachmentRef) bool {
	for _, ref := range refs {
		ext := strings.ToLower(filepath.Ext(ref.Filename))
		if ext == ".xml" || ext == ".pdf" || ext == ".zip" {
			return true
		}

		mime := strings.ToLower(strings.TrimSpace(ref.MimeType))
		if strings.Contains(mime, "pdf") || strings.Contains(mime, "xml") || strings.Contains(mime, "zip") {
			return true
		}
	}

	return false
}

func hasInvoiceKeyword(subject, body string) bool {
	combined := strings.ToLower(strings.TrimSpace(subject + "\n" + body))
	if combined == "" {
		return false
	}

	keywords := []string{
		"factura electronica",
		"facturación electrónica",
		"factura electrónica",
		"facturacion electronica",
		"facturacion",
		"factura",
		"invoice",
	}

	for _, keyword := range keywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	return false
}
