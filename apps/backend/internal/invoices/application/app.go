package application

import (
	"github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/invoices/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	CreateInvoicesFromInboxMessage  *commands.CreateInvoicesFromInboxMessageCommand
	QueueInvoiceExtractionFromFiles *commands.QueueInvoiceExtractionFromFilesCommand
	ProcessInvoiceExtractionJob     *commands.ProcessInvoiceExtractionJobCommand
	CreateInvoice                   *commands.CreateInvoiceCommand
}

type Queries struct {
	GetInvoiceByID queries.GetInvoiceByIDQuery
}
