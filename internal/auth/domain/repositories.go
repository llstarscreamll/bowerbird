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
	GetByID(ctx context.Context, ID string) (Session, error)
	Delete(ctx context.Context, sessionID string) error
}

type MailCredentialRepository interface {
	Save(ctx context.Context, ID, userID, walletID, mailProvider, mailAddress, accessToken, refreshToken string, expiresAt time.Time) error
	FindByWalletID(ctx context.Context, userID string) ([]MailCredential, error)
	Delete(ctx context.Context, ID string) error
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
	GetByWalletIDAndID(ctx context.Context, walletID, transactionID string) (Transaction, error)
	Update(ctx context.Context, transaction Transaction) error
}

type CategoryRepository interface {
	FindByWalletID(ctx context.Context, walletID string) ([]Category, error)
	Create(ctx context.Context, category Category) error
}

type FilePasswordRepository interface {
	GetByUserID(ctx context.Context, userID string) ([]string, error)
	Upsert(ctx context.Context, userID string, passwords []string) error
}
