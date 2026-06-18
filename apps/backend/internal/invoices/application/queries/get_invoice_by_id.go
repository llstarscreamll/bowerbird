package queries

import (
	"context"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
)

type GetInvoiceByIDQuery struct {
	repo ports.InvoiceQueryRepository
}

func NewGetInvoiceByIDQuery(repo ports.InvoiceQueryRepository) *GetInvoiceByIDQuery {
	return &GetInvoiceByIDQuery{repo: repo}
}

type InvoiceDetails struct {
	Header domain.InvoiceHeaderRecord
	Lines  []domain.InvoiceLineRecord
}

func (q *GetInvoiceByIDQuery) Execute(ctx context.Context, id string) (*InvoiceDetails, error) {
	header, lines, err := q.repo.GetInvoiceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &InvoiceDetails{
		Header: *header,
		Lines:  lines,
	}, nil
}
