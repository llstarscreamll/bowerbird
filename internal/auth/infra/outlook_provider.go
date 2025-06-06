package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
	"slices"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
	"google.golang.org/api/gmail/v1"
)

type OutlookProvider struct {
	config *oauth2.Config
	ulid   commonDomain.ULIDGenerator
}

func (g OutlookProvider) SearchByDateRangeAndSenders(ctx context.Context, tokens domain.Tokens, startDate time.Time, senders []string) ([]domain.MailMessage, error) {
	var result []domain.MailMessage

	url := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/me/messages?$filter=from/emailAddress/address eq '%s' AND receivedDateTime ge %s&$top=100",
		strings.Join(senders, "' OR from/emailAddress/address eq '"),
		startDate.Format(time.RFC3339),
	)

	for {
		if url == "" {
			break
		}

		messages, nextLink, err := g.getMessages(ctx, url, tokens)
		if err != nil {
			return nil, err
		}

		result = slices.Concat(result, messages)

		url = nextLink
	}

	return result, nil
}

func (g OutlookProvider) getMessages(ctx context.Context, url string, tokens domain.Tokens) ([]domain.MailMessage, string, error) {
	var result []domain.MailMessage

	httpClient := g.config.Client(ctx, &oauth2.Token{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Expiry:       tokens.ExpiresAt,
	})

	resp, err := httpClient.Get(strings.ReplaceAll(url, " ", "%20"))

	if err != nil {
		return nil, "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("failed to get outlook messages: %s", resp.Status)
	}

	var response messagesResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", err
	}

	for _, message := range response.Value {
		attachments := []domain.MailAttachment{}

		if message.HasAttachments {
			attachments, err = g.getAttachments(ctx, message.ID, tokens)

			if err != nil {
				fmt.Printf("error getting attachments: %s\n", err)
			}
		}

		result = append(result, domain.MailMessage{
			ID:          g.ulid.New(),
			ExternalID:  message.ID,
			From:        message.From.EmailAddress.Address,
			To:          message.ToRecipients[0].EmailAddress.Address,
			Subject:     message.Subject,
			Body:        message.Body.Content,
			ReceivedAt:  message.ReceivedDateTime,
			Attachments: attachments,
		})
	}

	return result, response.NextLink, nil
}

func (g OutlookProvider) getAttachments(ctx context.Context, messageID string, tokens domain.Tokens) ([]domain.MailAttachment, error) {
	httpClient := g.config.Client(ctx, &oauth2.Token{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Expiry:       tokens.ExpiresAt,
	})

	resp, err := httpClient.Get(fmt.Sprintf("https://graph.microsoft.com/v1.0/me/messages/%s/attachments", messageID))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("failed to get outlook attachments: %s", resp.Status)
		return nil, fmt.Errorf("failed to get outlook attachments: %s", resp.Status)
	}

	var response attachmentsResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	result := []domain.MailAttachment{}

	for _, attachment := range response.Value {
		if !strings.HasPrefix(attachment.ContentType, "application/pdf") {
			continue
		}

		if attachment.ContentBytes == "" {
			continue
		}

		result = append(result, domain.MailAttachment{
			Name:        attachment.Name,
			ContentType: attachment.ContentType,
			Content:     attachment.ContentBytes,
			Password:    attachment.ContentBytes,
		})
	}

	return result, nil
}

func (g OutlookProvider) Name() string {
	return "microsoft"
}

type attachmentsResponse struct {
	Value []struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		ContentType     string `json:"contentType"`
		Size            int    `json:"size"`
		IsInline        bool   `json:"isInline"`
		ContentId       string `json:"contentId"`
		ContentLocation string `json:"contentLocation"`
		ContentBytes    string `json:"contentBytes"`
	} `json:"value"`
}

type messagesResponse struct {
	Value []struct {
		ID   string `json:"id"`
		From struct {
			EmailAddress struct {
				Address string `json:"address"`
				Name    string `json:"name"`
			} `json:"emailAddress"`
		} `json:"from"`
		ToRecipients []struct {
			EmailAddress struct {
				Address string `json:"address"`
				Name    string `json:"name"`
			} `json:"emailAddress"`
		} `json:"toRecipients"`
		Subject string `json:"subject"`
		Body    struct {
			Content     string `json:"content"`
			ContentType string `json:"contentType"`
		} `json:"body"`
		HasAttachments       bool      `json:"hasAttachments"`
		IsRead               bool      `json:"isRead"`
		ReceivedDateTime     time.Time `json:"receivedDateTime"`
		CreatedDateTime      time.Time `json:"createdDateTime"`
		LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
		SentDateTimeDateTime time.Time `json:"sentDateTimeDateTime"`
	} `json:"value"`
	NextLink string `json:"@odata.nextLink"`
}

func NewOutlookProvider(clientID, clientSecret, redirectUrl string, ulid commonDomain.ULIDGenerator) *OutlookProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Endpoint:     microsoft.AzureADEndpoint(""),
		Scopes:       []string{gmail.GmailReadonlyScope},
	}

	return &OutlookProvider{config: config, ulid: ulid}
}
