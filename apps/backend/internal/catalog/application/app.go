package application

import (
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	ResolveInvoiceLine *commands.ResolveInvoiceLineCommand
	RememberDecision   *commands.RememberDecisionCommand
	LinkInvoiceLine    *commands.LinkInvoiceLineCommand
}

type Queries struct {
	GetItemByID     *queries.GetItemByIDQuery
	GetItemNames    *queries.GetItemNamesQuery
	ListItems       *queries.ListItemsQuery
	ListReviewQueue *queries.ListReviewQueueQuery
}
