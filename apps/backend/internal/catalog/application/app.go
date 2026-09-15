package application

import (
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/ports"
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
	AddItemAlias                *commands.AddItemAliasCommand
	RemoveItemAlias             *commands.RemoveItemAliasCommand
	RememberDecision            *commands.RememberDecisionCommand
	MergeItems                  *commands.MergeItemsCommand
	MarkNotDuplicates           *commands.MarkNotDuplicatesCommand
	QueueCatalogImport          *commands.QueueCatalogImportCommand
	ProcessCatalogImport        *commands.ProcessCatalogImportCommand
	CancelCatalogImport         *commands.CancelCatalogImportCommand
	PurgeStaleCatalogImports    *commands.PurgeStaleCatalogImportsCommand
}

type Queries struct {
	GetItemByID           *queries.GetItemByIDQuery
	GetItemNames          *queries.GetItemNamesQuery
	GetItemDisplays       *queries.GetItemDisplaysQuery
	ListItems             *queries.ListItemsQuery
	ListDuplicateClusters *queries.ListDuplicateClustersQuery
	GetImportByID         *queries.GetImportByIDQuery
	GetActiveImport       *queries.GetActiveImportQuery
	ListImports           *queries.ListImportsQuery
	ListImportErrors      *queries.ListImportErrorsQuery
}

func (a *Application) BindItemLinks(links ports.ItemLinkSupport) {
	if a == nil {
		panic("catalog application is required")
	}
	if links == nil {
		panic("item link support is required")
	}
	if a.Commands.MergeItems == nil {
		panic("merge items command is required")
	}
	if a.Queries.ListDuplicateClusters == nil {
		panic("list duplicate clusters query is required")
	}
	a.Commands.MergeItems.BindLinks(links)
	a.Queries.ListDuplicateClusters.BindLinks(links)
}
