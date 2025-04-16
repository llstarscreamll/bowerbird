package infra

import (
	"context"
	"encoding/base64"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GMailProvider struct {
	config *oauth2.Config
	ulid   commonDomain.ULIDGenerator
}

func (g GMailProvider) SearchByDateRangeAndSenders(ctx context.Context, tokens domain.Tokens, startDate time.Time, senders []string) ([]domain.MailMessage, error) {
	var result []domain.MailMessage

	oauthToken := &oauth2.Token{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, Expiry: tokens.ExpiresAt}
	tokenSource := g.config.TokenSource(ctx, oauthToken)
	token, err := tokenSource.Token()

	if err != nil {
		fmt.Println("Here!!")
		return nil, err
	}

	client := g.config.Client(ctx, token)
	mail, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	mailIDs, err := g.listMessages(mail, startDate, senders)
	if err != nil {
		return nil, err
	}

	for _, mailID := range mailIDs {
		message, err := g.getMessageDetail(mail, mailID)
		if err != nil {
			return nil, err
		}

		result = append(result, message)
	}

	return result, nil
}

func (g GMailProvider) listMessages(mail *gmail.Service, startDate time.Time, senders []string) ([]string, error) {
	var result []string
	from := strings.Join(senders, " OR from:")
	query := fmt.Sprintf("after:%s AND from:%s", startDate.Format("01/02/2006"), from)

	messagesList, err := mail.Users.Messages.
		List("me").
		IncludeSpamTrash(true).
		Q(query).
		Do()
	if err != nil {
		return nil, err
	}

	for _, m := range messagesList.Messages {
		result = append(result, m.Id)
	}

	return result, nil
}

func (g GMailProvider) getMessageDetail(mail *gmail.Service, messageID string) (domain.MailMessage, error) {
	message := domain.MailMessage{}

	msg, err := mail.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return message, err
	}

	message.ID = g.ulid.New()
	message.ExternalID = msg.Id

	for _, h := range msg.Payload.Headers {
		if strings.ToLower(h.Name) == "from" {
			message.From = h.Value
		}

		if strings.ToLower(h.Name) == "to" {
			message.To = h.Value
		}

		if strings.ToLower(h.Name) == "subject" {
			message.Subject = h.Value
		}

		if strings.ToLower(h.Name) == "date" {
			h.Value = strings.TrimSuffix(h.Value, " (COT)") // date has format: Wed, 29 Jan 2025 15:20:41 -0500 (COT)
			date, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", h.Value)
			if err != nil {
				return domain.MailMessage{}, err
			}

			message.ReceivedAt = date
		}
	}

	decodedBody, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
	if err != nil {
		return domain.MailMessage{}, err
	}

	message.Body = string(decodedBody)

	return message, nil
}

func (g GMailProvider) Name() string {
	return "google"
}

func NewGMailProvider(clientID, clientSecret, redirectUrl string, ulid commonDomain.ULIDGenerator) *GMailProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailReadonlyScope},
	}

	return &GMailProvider{config: config, ulid: ulid}
}
