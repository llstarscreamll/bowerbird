package infra

import (
	"context"
	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

type OutlookProvider struct {
	config *oauth2.Config
	ulid   commonDomain.ULIDGenerator
}

func (g OutlookProvider) SearchByDateRangeAndSenders(ctx context.Context, tokens domain.Tokens, startDate time.Time, senders []string) ([]domain.MailMessage, error) {
	var result []domain.MailMessage

	return result, nil
}

func (g OutlookProvider) listMessages(mail *gmail.Service, startDate time.Time, senders []string) ([]string, error) {
	var result []string

	return result, nil
}

func (g OutlookProvider) getMessageDetail(mail *gmail.Service, messageID string) (domain.MailMessage, error) {
	message := domain.MailMessage{}

	return message, nil
}

func (g OutlookProvider) Name() string {
	return "microsoft"
}

func NewOutlookProvider(clientID, clientSecret, redirectUrl string, ulid commonDomain.ULIDGenerator) *OutlookProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailReadonlyScope},
	}

	return &OutlookProvider{config: config, ulid: ulid}
}
