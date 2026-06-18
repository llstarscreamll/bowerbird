package queries

import (
	"context"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
)

type ListInvoicesQuery struct {
	repo ports.InvoiceQueryRepository
}

func NewListInvoicesQuery(repo ports.InvoiceQueryRepository) *ListInvoicesQuery {
	return &ListInvoicesQuery{repo: repo}
}

type PaginatedInvoices struct {
	Items   []domain.InvoiceHeaderRecord
	HasMore bool
	Limit   int
	Cursor  string
}

func (q *ListInvoicesQuery) Execute(ctx context.Context, limit int, cursor string) (*PaginatedInvoices, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	headers, hasMore, err := q.repo.ListInvoices(ctx, limit, cursor)
	if err != nil {
		return nil, err
	}

	var nextCursor string
	if len(headers) > 0 {
		nextCursor = headers[len(headers)-1].ID
	}

	return &PaginatedInvoices{
		Items:   headers,
		HasMore: hasMore,
		Limit:   limit,
		Cursor:  nextCursor,
	}, nil
}
