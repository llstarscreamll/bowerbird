package infra

import (
	"context"
	"errors"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"time"
)

type MailGateway struct {
	strategies map[string]domain.MailProvider
}

func (g MailGateway) SearchFromDateAndSenders(ctx context.Context, provider string, tokens domain.Tokens, startDate time.Time, senders []string) ([]domain.MailMessage, error) {
	p, ok := g.strategies[provider]
	if !ok {
		return nil, errors.New("Unsupported mail provider " + provider)
	}

	return p.SearchByDateRangeAndSenders(ctx, tokens, startDate, senders)
}

func NewMailGateway(googleMail domain.MailProvider) *MailGateway {
	return &MailGateway{strategies: map[string]domain.MailProvider{
		"google": googleMail,
	}}
}
