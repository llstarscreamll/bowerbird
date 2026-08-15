package application

import (
	"github.com/bowerbird/internal/inbox/application/commands"
	"github.com/bowerbird/internal/inbox/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	SyncAccount        *commands.SyncAccountCommand
	SyncAllAccounts    *commands.SyncAllAccountsCommand
	ModifyMessage      *commands.ModifyMessageCommand
	SendMessage        *commands.SendMessageCommand
	DownloadAttachment *commands.DownloadAttachmentCommand
}

type Queries struct {
	ListAccountHealth *queries.ListAccountHealthQuery
	ListMessages      *queries.ListMessagesQuery
	GetMessage        *queries.GetMessageQuery
}
