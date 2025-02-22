package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	Upsert(ctx context.Context, user User) (string, error)
	GetByID(ctx context.Context, ID string) (User, error)
}

type SessionRepository interface {
	Save(ctx context.Context, userID, sessionID string, expirationDate time.Time) error
	GetByID(ctx context.Context, ID string) (string, error)
	Delete(ctx context.Context, sessionID string) error
}

type MailCredentialRepository interface {
	Save(ctx context.Context, ID, userID, walletID, mailProvider, mailAddress, accessToken, refreshToken string, expiresAt time.Time) error
	FindByUserID(ctx context.Context, userID string) ([]MailCredential, error)
}

type MailMessageRepository interface {
	UpsertMany(ctx context.Context, messages []MailMessage) error
}

type WalletRepository interface {
	Create(ctx context.Context, wallet UserWallet) error
	FindByUserID(ctx context.Context, userID string) ([]UserWallet, error)
}

type TransactionRepository interface {
	UpsertMany(ctx context.Context, transactions []Transaction) error
	FindByWalletID(ctx context.Context, walletID string) ([]Transaction, error)
}
