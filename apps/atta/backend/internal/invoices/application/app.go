package application

import (
	invoicesapi "github.com/atta/internal/invoices/api"
	"github.com/atta/internal/invoices/application/commands"
	"github.com/atta/internal/invoices/application/queries"
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
