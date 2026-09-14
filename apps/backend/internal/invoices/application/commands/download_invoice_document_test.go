package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/bowerbird/internal/invoices/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type downloadInvoiceRepoStub struct {
	header *domain.InvoiceHeaderRecord
	err    error
}

func (s *downloadInvoiceRepoStub) GetInvoiceByID(ctx context.Context, id string) (*domain.InvoiceHeaderRecord, []domain.InvoiceLineRecord, error) {
	if s.err != nil {
		return nil, nil, s.err
	}
	if s.header == nil || s.header.ID != id {
		return nil, nil, errors.New("invoice not found")
	}
	cp := *s.header
	return &cp, nil, nil
}

func (s *downloadInvoiceRepoStub) ListInvoices(ctx context.Context, limit int, cursor string) ([]domain.InvoiceHeaderRecord, bool, error) {
	return nil, false, nil
}

type downloadInvoiceStoreStub struct {
	data map[string][]byte
}

func (s *downloadInvoiceStoreStub) WriteFileIfAbsent(ctx context.Context, input platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	return nil, errors.New("not implemented")
}

func (s *downloadInvoiceStoreStub) ReadFile(ctx context.Context, input platformStorage.ReadFileInput) ([]byte, error) {
	payload, ok := s.data[input.Path]
	if !ok {
		return nil, errors.New("not found")
	}
	return payload, nil
}

func (s *downloadInvoiceStoreStub) DownloadFile(ctx context.Context, input platformStorage.DownloadFileInput) error {
	return errors.New("not implemented")
}

func (s *downloadInvoiceStoreStub) Exists(ctx context.Context, input platformStorage.ExistsFileInput) (bool, error) {
	_, ok := s.data[input.Path]
	return ok, nil
}

func (s *downloadInvoiceStoreStub) MoveFile(ctx context.Context, input platformStorage.MoveFileInput) error {
	return nil
}

func (s *downloadInvoiceStoreStub) PresignUpload(ctx context.Context, input platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	return nil, errors.New("not implemented")
}

func (s *downloadInvoiceStoreStub) PresignDownload(ctx context.Context, input platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
	return nil, errors.New("not implemented")
}

func TestDownloadInvoiceDocumentReturnsOriginalZip(t *testing.T) {
	key := "tenant/t1/inbox/raw/2026/05/25/msg/fv123.zip"
	cmd := NewDownloadInvoiceDocumentCommand(
		&downloadInvoiceRepoStub{header: &domain.InvoiceHeaderRecord{ID: "INV-1", DocumentRefS3Key: key}},
		&downloadInvoiceStoreStub{data: map[string][]byte{key: []byte("PK zip")}},
	)

	result, err := cmd.Execute(context.Background(), "INV-1")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "fv123.zip", result.Filename)
	assert.Equal(t, "application/zip", result.ContentType)
	assert.Equal(t, []byte("PK zip"), result.Data)
}

func TestDownloadInvoiceDocumentReturnsStandaloneXML(t *testing.T) {
	key := "1-day/t1/uploads/user/factura.xml"
	cmd := NewDownloadInvoiceDocumentCommand(
		&downloadInvoiceRepoStub{header: &domain.InvoiceHeaderRecord{ID: "INV-2", DocumentRefS3Key: key}},
		&downloadInvoiceStoreStub{data: map[string][]byte{key: []byte("<Invoice/>")}},
	)

	result, err := cmd.Execute(context.Background(), "INV-2")

	require.NoError(t, err)
	assert.Equal(t, "factura.xml", result.Filename)
	assert.Equal(t, "application/xml", result.ContentType)
	assert.Equal(t, []byte("<Invoice/>"), result.Data)
}

func TestDownloadInvoiceDocumentNotFound(t *testing.T) {
	cmd := NewDownloadInvoiceDocumentCommand(
		&downloadInvoiceRepoStub{},
		&downloadInvoiceStoreStub{data: map[string][]byte{}},
	)

	_, err := cmd.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, ErrInvoiceNotFound)
}

func TestDownloadInvoiceDocumentMissingStorageKey(t *testing.T) {
	cmd := NewDownloadInvoiceDocumentCommand(
		&downloadInvoiceRepoStub{header: &domain.InvoiceHeaderRecord{ID: "INV-3"}},
		&downloadInvoiceStoreStub{data: map[string][]byte{}},
	)

	_, err := cmd.Execute(context.Background(), "INV-3")

	require.ErrorIs(t, err, ErrInvoiceDocumentNotFound)
}
