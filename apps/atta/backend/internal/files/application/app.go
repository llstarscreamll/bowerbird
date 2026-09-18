package application

import (
	"github.com/atta/internal/files/application/commands"
	platformStorage "github.com/atta/internal/platform/storage"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	RequestUploadURL   *commands.RequestUploadURLCommand
	RequestDownloadURL *commands.RequestDownloadURLCommand
}

type Queries struct{}

func NewApplication(fileStore platformStorage.FileStore) *Application {
	return &Application{
		Commands: Commands{
			RequestUploadURL:   commands.NewRequestUploadURLCommand(fileStore),
			RequestDownloadURL: commands.NewRequestDownloadURLCommand(fileStore),
		},
	}
}
