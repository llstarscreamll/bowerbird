package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bowerbird/internal/inbox/domain"
)

const defaultBaseURL = "https://graph.microsoft.com"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

var _ domain.MailProviderClient = (*Client)(nil)

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient, baseURL: defaultBaseURL}
}

func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = strings.TrimRight(baseURL, "/")
}

func (c *Client) ListMessages(ctx context.Context, opts domain.ListMessagesOptions) ([]domain.MessageRef, string, error) {
	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	values := url.Values{}
	values.Set("$top", fmt.Sprintf("%d", maxResults))
	values.Set("$select", "id,conversationId")
	if opts.Query != "" {
		values.Set("$filter", opts.Query)
	}
	if opts.PageToken != "" {
		values.Set("$skiptoken", opts.PageToken)
	}

	endpoint := fmt.Sprintf("%s/v1.0/me/messages?%s", c.baseURL, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", requestStatusError("list messages", resp)
	}

	var payload struct {
		Value []struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversationId"`
		} `json:"value"`
		NextLink string `json:"@odata.nextLink"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, "", err
	}

	refs := make([]domain.MessageRef, 0, len(payload.Value))
	for _, item := range payload.Value {
		refs = append(refs, domain.MessageRef{ID: item.ID, ThreadID: item.ConversationID})
	}

	next := ""
	if payload.NextLink != "" {
		if parsed, err := url.Parse(payload.NextLink); err == nil {
			next = parsed.Query().Get("$skiptoken")
		}
	}
	return refs, next, nil
}

func (c *Client) GetMessage(ctx context.Context, userID, messageID string) (*domain.MailMessage, error) {
	endpoint := fmt.Sprintf("%s/v1.0/me/messages/%s?$expand=attachments", c.baseURL, url.PathEscape(messageID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, requestStatusError("get message", resp)
	}

	var payload graphMessage
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.toMailMessage(), nil
}

func (c *Client) DownloadAttachment(ctx context.Context, userID, messageID, attachmentID string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/v1.0/me/messages/%s/attachments/%s/$value", c.baseURL, url.PathEscape(messageID), url.PathEscape(attachmentID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, requestStatusError("download attachment", resp)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 128*1024*1024))
}

func (c *Client) DownloadMessageAttachments(ctx context.Context, userID, messageID string, refs []domain.MailAttachmentRef) ([]domain.DownloadedMailAttachment, error) {
	results := make([]domain.DownloadedMailAttachment, 0, len(refs))
	for _, ref := range refs {
		data, err := c.DownloadAttachment(ctx, userID, messageID, ref.AttachmentID)
		if err != nil {
			return nil, err
		}
		results = append(results, domain.DownloadedMailAttachment{MailAttachmentRef: ref, Data: data})
	}
	return results, nil
}

func (c *Client) CreateLabel(ctx context.Context, userID, labelName string) (string, error) {
	body, _ := json.Marshal(map[string]string{"displayName": labelName})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1.0/me/mailFolders", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", requestStatusError("create folder", resp)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func (c *Client) AddLabelToMessage(ctx context.Context, userID, messageID, labelID string) error {
	body, _ := json.Marshal(map[string]string{"destinationId": labelID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1.0/me/messages/%s/move", c.baseURL, url.PathEscape(messageID)), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return requestStatusError("move message", resp)
	}
	return nil
}

func (c *Client) GetHistoryID(ctx context.Context, userID string) (string, error) {
	return "", nil
}

func (c *Client) ListHistory(ctx context.Context, userID, startHistoryID string) (domain.HistoryPage, error) {
	return domain.HistoryPage{Expired: true}, nil
}

func (c *Client) ModifyMessage(ctx context.Context, userID, messageID string, mutation domain.MessageMutation) error {
	patch := map[string]any{}
	for _, label := range mutation.RemoveLabelIDs {
		if label == "UNREAD" {
			patch["isRead"] = true
		}
		if label == "STARRED" {
			patch["flag"] = map[string]string{"flagStatus": "notFlagged"}
		}
		if label == "INBOX" {
			return c.AddLabelToMessage(ctx, userID, messageID, "archive")
		}
	}
	for _, label := range mutation.AddLabelIDs {
		if label == "UNREAD" {
			patch["isRead"] = false
		}
		if label == "STARRED" {
			patch["flag"] = map[string]string{"flagStatus": "flagged"}
		}
	}
	if len(patch) == 0 {
		return nil
	}
	body, _ := json.Marshal(patch)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, fmt.Sprintf("%s/v1.0/me/messages/%s", c.baseURL, url.PathEscape(messageID)), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return requestStatusError("modify message", resp)
	}
	return nil
}

func (c *Client) TrashMessage(ctx context.Context, userID, messageID string) error {
	return c.AddLabelToMessage(ctx, userID, messageID, "deleteditems")
}

func (c *Client) SendMessage(ctx context.Context, userID string, message domain.OutgoingMail) (string, error) {
	if len(message.To) == 0 {
		return "", domain.ErrOutgoingMailToRequired
	}
	contentType := "Text"
	content := message.BodyText
	if strings.TrimSpace(message.BodyHTML) != "" {
		contentType = "HTML"
		content = message.BodyHTML
	}
	payload := map[string]any{
		"message": map[string]any{
			"subject":       message.Subject,
			"body":          map[string]string{"contentType": contentType, "content": content},
			"toRecipients":  graphRecipients(message.To),
			"ccRecipients":  graphRecipients(message.Cc),
			"bccRecipients": graphRecipients(message.Bcc),
		},
		"saveToSentItems": true,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1.0/me/sendMail", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", requestStatusError("send mail", resp)
	}
	return fmt.Sprintf("graph-%d", time.Now().UTC().UnixNano()), nil
}

type graphMessage struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	Subject        string `json:"subject"`
	From           *struct {
		EmailAddress struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		} `json:"emailAddress"`
	} `json:"from"`
	ToRecipients  []graphRecipient `json:"toRecipients"`
	CcRecipients  []graphRecipient `json:"ccRecipients"`
	BccRecipients []graphRecipient `json:"bccRecipients"`
	BodyPreview   string           `json:"bodyPreview"`
	Body          *struct {
		ContentType string `json:"contentType"`
		Content     string `json:"content"`
	} `json:"body"`
	ReceivedDateTime time.Time `json:"receivedDateTime"`
	IsRead           bool      `json:"isRead"`
	IsDraft          bool      `json:"isDraft"`
	Flag             *struct {
		FlagStatus string `json:"flagStatus"`
	} `json:"flag"`
	ParentFolderID string `json:"parentFolderId"`
	Attachments    []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
		IsInline    bool   `json:"isInline"`
	} `json:"attachments"`
}

type graphRecipient struct {
	EmailAddress struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	} `json:"emailAddress"`
}

func (m graphMessage) toMailMessage() *domain.MailMessage {
	sender := ""
	if m.From != nil {
		sender = m.From.EmailAddress.Address
		if m.From.EmailAddress.Name != "" {
			sender = m.From.EmailAddress.Name + " <" + m.From.EmailAddress.Address + ">"
		}
	}
	received := m.ReceivedDateTime.UTC()
	labels := graphLabels(m)
	msg := &domain.MailMessage{
		ID:         m.ID,
		ThreadID:   m.ConversationID,
		LabelIDs:   labels,
		Subject:    m.Subject,
		Sender:     sender,
		To:         recipientAddresses(m.ToRecipients),
		Cc:         recipientAddresses(m.CcRecipients),
		Bcc:        recipientAddresses(m.BccRecipients),
		Snippet:    m.BodyPreview,
		ReceivedAt: &received,
	}
	if m.Body != nil {
		if strings.EqualFold(m.Body.ContentType, "html") {
			msg.HTMLBody = m.Body.Content
		} else {
			msg.PlainTextBody = m.Body.Content
		}
	}
	for _, att := range m.Attachments {
		if att.IsInline {
			continue
		}
		msg.Attachments = append(msg.Attachments, domain.MailAttachmentRef{
			AttachmentID: att.ID,
			Filename:     att.Name,
			MimeType:     att.ContentType,
			Size:         att.Size,
		})
	}
	return msg
}

func graphLabels(m graphMessage) []string {
	labels := []string{"INBOX"}
	if !m.IsRead {
		labels = append(labels, "UNREAD")
	}
	if m.IsDraft {
		labels = append(labels, "DRAFT")
	}
	if m.Flag != nil && strings.EqualFold(m.Flag.FlagStatus, "flagged") {
		labels = append(labels, "STARRED")
	}
	return labels
}

func recipientAddresses(recipients []graphRecipient) []string {
	out := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		if recipient.EmailAddress.Address != "" {
			out = append(out, recipient.EmailAddress.Address)
		}
	}
	return out
}

func graphRecipients(addresses []string) []map[string]any {
	out := make([]map[string]any, 0, len(addresses))
	for _, address := range addresses {
		out = append(out, map[string]any{"emailAddress": map[string]string{"address": address}})
	}
	return out
}

func requestStatusError(prefix string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("%s failed with status %d: %s", prefix, resp.StatusCode, strings.TrimSpace(string(body)))
}
