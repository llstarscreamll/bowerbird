package application

import "github.com/bowerbird/internal/connections/application/commands"

var ErrCipherNotConfigured = commands.ErrCipherNotConfigured

type CredentialsCipher = commands.CredentialsCipher
type CredentialsService = commands.CredentialsService

var NewCredentialsService = commands.NewCredentialsService
