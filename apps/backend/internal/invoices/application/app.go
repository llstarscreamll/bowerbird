package application

import (
	invoicesapi "github.com/bowerbird/internal/invoices/api"
	"github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/invoices/application/queries"
)

type Application struct {
	Commands  Commands
	Queries   Queries
	ItemLinks invoicesapi.ItemLinkSupport
}

type Commands struct {
	CreateInvoicesFromInboxMessage  *commands.CreateInvoicesFromInboxMessageCommand
	QueueInvoiceExtractionFromFiles *commands.QueueInvoiceExtractionFromFilesCommand
	ProcessInvoiceExtractionJob     *commands.CreateInvoicesFromFilesCommand
	CreateInvoice                   *commands.CreateInvoiceCommand
	ApplyLineDecision               *commands.ApplyLineDecisionCommand
	DownloadInvoiceDocument         *commands.DownloadInvoiceDocumentCommand
	BackfillInboxInvoices           *commands.BackfillInboxInvoicesCommand
}

type Queries struct {
	GetInvoiceByID  *queries.GetInvoiceByIDQuery
	ListInvoices    *queries.ListInvoicesQuery
	ListReviewQueue *queries.ListReviewQueueQuery
}
