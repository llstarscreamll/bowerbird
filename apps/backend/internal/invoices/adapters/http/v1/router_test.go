package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bowerbird/internal/invoices/application"
	"github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/application/queries"
	"github.com/bowerbird/internal/invoices/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLineRepo struct {
	lines map[string]*domain.LineForDecision
}

func (s *stubLineRepo) SaveLineLink(ctx context.Context, lineID string, link domain.LineLink) error {
	line := s.lines[lineID]
	if line != nil {
		line.Link = link
	}
	return nil
}
func (s *stubLineRepo) ListReviewLines(ctx context.Context, statuses []string) ([]ports.ReviewLine, error) {
	return []ports.ReviewLine{{
		LineID:          "LINE-1",
		InvoiceHeaderID: "INV-1",
		Description:     "Widget",
		LinkStatus:      domain.LinkStatusUnmatched,
	}}, nil
}
func (s *stubLineRepo) GetLineForDecision(ctx context.Context, lineID string) (*domain.LineForDecision, error) {
	line, ok := s.lines[lineID]
	if !ok {
		return nil, nil
	}
	cp := *line
	return &cp, nil
}
func (s *stubLineRepo) SyncHeaderLinkingStatus(ctx context.Context, invoiceHeaderID string) error {
	return nil
}

type stubCatalogSvc struct{}

func (s *stubCatalogSvc) GetItemNames(ctx context.Context, ids []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *stubCatalogSvc) GetItemDisplays(ctx context.Context, ids []string) (map[string]ports.ItemDisplay, error) {
	return map[string]ports.ItemDisplay{}, nil
}

type stubMatching struct{}

func (s *stubMatching) ValidateItemExists(ctx context.Context, itemID string) error { return nil }
func (s *stubMatching) MintProvisionalFromEvidence(ctx context.Context, input ports.MintProvisionalInput) (string, error) {
	return "ITEM-NEW", nil
}
func (s *stubMatching) RememberDecision(ctx context.Context, input ports.RememberDecisionInput) error {
	return nil
}

func TestApplyLineDecisionHTTP(t *testing.T) {
	repo := &stubLineRepo{
		lines: map[string]*domain.LineForDecision{
			"LINE-1": {
				LineID:          "LINE-1",
				InvoiceHeaderID: "INV-1",
				Link:            domain.LineLink{Status: domain.LinkStatusUnmatched},
			},
		},
	}
	app := &application.Application{
		Commands: application.Commands{
			ApplyLineDecision: commands.NewApplyLineDecisionCommand(repo, &stubMatching{}),
		},
		Queries: application.Queries{
			ListReviewQueue: queries.NewListReviewQueueQuery(repo, &stubCatalogSvc{}),
		},
	}
	ctrl := NewController(app)

	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type": "invoice_line_decisions",
			"attributes": map[string]any{
				"item_id":  "ITEM-1",
				"action":   "link",
				"remember": true,
				"lock":     true,
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoicing/invoices/INV-1/lines/LINE-1/decisions", bytes.NewReader(body))
	req.SetPathValue("invoiceId", "INV-1")
	req.SetPathValue("lineId", "LINE-1")
	rr := httptest.NewRecorder()
	err := ctrl.ApplyLineDecision(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Equal(t, domain.LinkStatusLinked, repo.lines["LINE-1"].Link.Status)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoicing/review-queue", nil)
	listRR := httptest.NewRecorder()
	err = ctrl.ListReviewQueue(listRR, listReq)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, listRR.Code)
}

type stubInvoiceQueryRepo struct {
	header *domain.InvoiceHeaderRecord
}

func (s *stubInvoiceQueryRepo) GetInvoiceByID(ctx context.Context, id string) (*domain.InvoiceHeaderRecord, []domain.InvoiceLineRecord, error) {
	if s.header == nil || s.header.ID != id {
		return nil, nil, errors.New("invoice not found")
	}
	cp := *s.header
	return &cp, nil, nil
}

func (s *stubInvoiceQueryRepo) ListInvoices(ctx context.Context, limit int, cursor string) ([]domain.InvoiceHeaderRecord, bool, error) {
	return nil, false, nil
}

type stubInvoiceFileStore struct {
	data map[string][]byte
}

func (s *stubInvoiceFileStore) WriteFileIfAbsent(ctx context.Context, input platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	return nil, errors.New("not implemented")
}

func (s *stubInvoiceFileStore) ReadFile(ctx context.Context, input platformStorage.ReadFileInput) ([]byte, error) {
	payload, ok := s.data[input.Path]
	if !ok {
		return nil, errors.New("not found")
	}
	return payload, nil
}

func (s *stubInvoiceFileStore) OpenFile(ctx context.Context, input platformStorage.OpenFileInput) (*platformStorage.OpenFileResult, error) {
	return nil, errors.New("not implemented")
}

func (s *stubInvoiceFileStore) DownloadFile(ctx context.Context, input platformStorage.DownloadFileInput) error {
	return errors.New("not implemented")
}

func (s *stubInvoiceFileStore) Exists(ctx context.Context, input platformStorage.ExistsFileInput) (bool, error) {
	_, ok := s.data[input.Path]
	return ok, nil
}

func (s *stubInvoiceFileStore) MoveFile(ctx context.Context, input platformStorage.MoveFileInput) error {
	return nil
}

func (s *stubInvoiceFileStore) PresignUpload(ctx context.Context, input platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	return nil, errors.New("not implemented")
}

func (s *stubInvoiceFileStore) PresignDownload(ctx context.Context, input platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
	return nil, errors.New("not implemented")
}

func TestDownloadInvoiceDocumentHTTP(t *testing.T) {
	key := "tenant/t1/inbox/fv123.zip"
	app := &application.Application{
		Commands: application.Commands{
			DownloadInvoiceDocument: commands.NewDownloadInvoiceDocumentCommand(
				&stubInvoiceQueryRepo{header: &domain.InvoiceHeaderRecord{ID: "INV-1", DocumentRefS3Key: key}},
				&stubInvoiceFileStore{data: map[string][]byte{key: []byte("PK zip")}},
			),
		},
	}
	ctrl := NewController(app)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoicing/invoices/INV-1/document", nil)
	req.SetPathValue("id", "INV-1")
	rr := httptest.NewRecorder()
	err := ctrl.DownloadInvoiceDocument(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"))
	assert.Equal(t, `attachment; filename="fv123.zip"`, rr.Header().Get("Content-Disposition"))
	assert.Equal(t, "PK zip", rr.Body.String())
}
