package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/id"
)

type PersistInvoiceInput struct {
	SourceMessageID  string
	ExtractionSource string
	DocumentRefS3Key string
	Invoice          *domain.InvoiceDocument
}

type PersistInvoiceResult struct {
	HeaderID string
	LineIDs  []string
}

type PersistInvoiceCommand struct {
	repo   domain.InvoiceWriteRepository
	logger *slog.Logger
	now    func() time.Time
	newID  func() string
}

func NewPersistInvoiceCommand(repo domain.InvoiceWriteRepository) *PersistInvoiceCommand {
	return &PersistInvoiceCommand{repo: repo, logger: slog.Default(), now: time.Now, newID: id.NewULID}
}

func (cmd *PersistInvoiceCommand) Execute(ctx context.Context, input PersistInvoiceInput) (*PersistInvoiceResult, error) {
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
	headerRawData := input.Invoice.RawData
	if len(headerRawData) == 0 {
		headerRawData = []byte("{}")
	}

	header := domain.InvoiceHeaderRecord{
		ID:               headerID,
		SourceMessageID:  input.SourceMessageID,
		CUFE:             input.Invoice.CUFE,
		InvoiceNumber:    input.Invoice.InvoiceID,
		IssuerName:       input.Invoice.Issuer.Name,
		IssuerTaxID:      input.Invoice.Issuer.CompanyID,
		ReceiverName:     input.Invoice.Receiver.Name,
		ReceiverTaxID:    input.Invoice.Receiver.CompanyID,
		CurrencyCode:     input.Invoice.CurrencyCode,
		IssueDate:        input.Invoice.IssueDateTimeUTC(),
		PaymentCode:      input.Invoice.PaymentMeansCode,
		Subtotal:         input.Invoice.LineExtension,
		TaxTotal:         input.Invoice.TaxAmountTotal(),
		GrandTotal:       input.Invoice.PayableAmount,
		DocumentRefS3Key: input.DocumentRefS3Key,
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
			ItemCode:        "",
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

	return &PersistInvoiceResult{HeaderID: headerID, LineIDs: lineIDs}, nil
}
