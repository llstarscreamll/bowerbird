package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
)

type InvoiceDedupRepository interface {
	ExistsInvoiceBySourceMessageID(ctx context.Context, sourceMessageID string) (bool, error)
	ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error)
}

type ExtractInvoiceStatus string

const (
	ExtractInvoiceStatusReady   ExtractInvoiceStatus = "ready"
	ExtractInvoiceStatusSkipped ExtractInvoiceStatus = "skipped"
)

type ExtractInvoiceSkipReason string

const (
	SkipReasonMessageAlreadyProcessed ExtractInvoiceSkipReason = "message_already_processed"
	SkipReasonNoSupportedDocument     ExtractInvoiceSkipReason = "no_supported_document"
	SkipReasonCUFEAlreadyExists       ExtractInvoiceSkipReason = "cufe_already_exists"
)

type ExtractInvoiceResult struct {
	Status     ExtractInvoiceStatus
	SkipReason ExtractInvoiceSkipReason
	Source     string
	Invoice    *domain.InvoiceDocument
}

type ExtractInvoiceUseCase struct {
	xmlExtractor domain.InvoiceXMLExtractor
	llmExtractor domain.InvoiceLLMExtractor
	repo         InvoiceDedupRepository
	logger       *slog.Logger
}

func NewExtractInvoiceUseCase(
	xmlExtractor domain.InvoiceXMLExtractor,
	llmExtractor domain.InvoiceLLMExtractor,
	repo InvoiceDedupRepository,
) *ExtractInvoiceUseCase {
	return &ExtractInvoiceUseCase{
		xmlExtractor: xmlExtractor,
		llmExtractor: llmExtractor,
		repo:         repo,
		logger:       slog.Default(),
	}
}

func (u *ExtractInvoiceUseCase) ExtractFromGroup(ctx context.Context, sourceMessageID string, group domain.DocumentGroup) (*ExtractInvoiceResult, error) {
	if u.repo == nil {
		return nil, fmt.Errorf("invoice dedup repository is required")
	}

	processed, err := u.repo.ExistsInvoiceBySourceMessageID(ctx, sourceMessageID)
	if err != nil {
		return nil, fmt.Errorf("check invoice by source message id: %w", err)
	}
	if processed {
		u.logger.Info("invoice extraction skipped by source message", "source_message_id", sourceMessageID)
		return &ExtractInvoiceResult{Status: ExtractInvoiceStatusSkipped, SkipReason: SkipReasonMessageAlreadyProcessed}, nil
	}

	invoice, source, err := u.extractInvoiceDocument(ctx, group)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return &ExtractInvoiceResult{Status: ExtractInvoiceStatusSkipped, SkipReason: SkipReasonNoSupportedDocument}, nil
	}

	duplicated, err := u.repo.ExistsInvoiceByCUFE(ctx, invoice.CUFE)
	if err != nil {
		return nil, fmt.Errorf("check invoice by cufe: %w", err)
	}
	if duplicated {
		u.logger.Info("invoice extraction skipped by cufe", "cufe", invoice.CUFE)
		return &ExtractInvoiceResult{Status: ExtractInvoiceStatusSkipped, SkipReason: SkipReasonCUFEAlreadyExists}, nil
	}

	u.logger.Info("invoice extracted and ready", "source", source, "cufe", invoice.CUFE)

	return &ExtractInvoiceResult{
		Status:  ExtractInvoiceStatusReady,
		Source:  source,
		Invoice: invoice,
	}, nil
}

func (u *ExtractInvoiceUseCase) extractInvoiceDocument(ctx context.Context, group domain.DocumentGroup) (*domain.InvoiceDocument, string, error) {
	if !group.SupportsInvoiceExtraction() {
		return nil, "", nil
	}

	source := group.PreferredDocumentSource()
	if source == "xml" {
		if u.xmlExtractor == nil {
			return nil, "", fmt.Errorf("xml extractor is required")
		}
		invoice, err := u.xmlExtractor.ParseInvoiceXML(group.XML.Data)
		if err != nil {
			return nil, "", fmt.Errorf("extract invoice from xml: %w", err)
		}
		return invoice, "xml", nil
	}

	if source == "llm" {
		if u.llmExtractor == nil {
			return nil, "", fmt.Errorf("llm extractor is required")
		}
		invoice, err := u.llmExtractor.ExtractFromPDF(ctx, group.PDF.Data)
		if err != nil {
			return nil, "", fmt.Errorf("extract invoice from pdf with llm: %w", err)
		}
		return invoice, "llm", nil
	}

	return nil, "", nil
}
