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
	ResolveInvoiceLine          *commands.ResolveInvoiceLineCommand
	ValidateCatalogItem         *commands.ValidateCatalogItemCommand
	MintProvisionalFromEvidence *commands.MintProvisionalFromEvidenceCommand
	EnsureSupplierAlias         *commands.EnsureSupplierAliasCommand
	RecordMatchMemory           *commands.RecordMatchMemoryCommand
	CreateItem                  *commands.CreateItemCommand
	UpdateItem                  *commands.UpdateItemCommand
}

type Queries struct {
	GetItemByID  *queries.GetItemByIDQuery
	GetItemNames *queries.GetItemNamesQuery
	ListItems    *queries.ListItemsQuery
}
