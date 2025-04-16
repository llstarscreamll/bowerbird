package domain

import (
	"context"
	"time"
)

type AuthServerGateway interface {
	GetLoginUrl(provider, redirectUrl string, scopes []string, state string) (string, error)
	GetTokens(ctx context.Context, provider string, authCode string, state string) (Tokens, error)
	GetUserProfile(ctx context.Context, provider string, authCode string) (User, error)
}

type AuthServerStrategy interface {
	GetLoginUrl(redirectUrl string, scopes []string, state string) (string, error)
	GetTokens(ctx context.Context, authCode, state string) (Tokens, error)
	GetUserProfile(ctx context.Context, authCode string) (User, error)
}

type MailGateway interface {
	SearchFromDateAndSenders(ctx context.Context, provider string, tokens Tokens, startDate time.Time, senders []string) ([]MailMessage, error)
}

type MailProvider interface {
	SearchByDateRangeAndSenders(ctx context.Context, tokens Tokens, startDate time.Time, senders []string) ([]MailMessage, error)
	Name() string
}
