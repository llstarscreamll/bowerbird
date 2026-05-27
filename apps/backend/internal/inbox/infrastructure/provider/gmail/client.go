package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

const defaultBaseURL = "https://gmail.googleapis.com"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

var _ domain.MailProviderClient = (*Client)(nil)

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    defaultBaseURL,
	}
}

func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = strings.TrimRight(baseURL, "/")
}

func (c *Client) ListMessages(ctx context.Context, opts domain.ListMessagesOptions) ([]domain.MessageRef, string, error) {
	userID := opts.UserID
	if userID == "" {
		userID = "me"
	}

	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	labelIDs := opts.LabelIDs
	if len(labelIDs) == 0 {
		labelIDs = []string{"UNREAD"}
	}

	values := url.Values{}
	values.Set("maxResults", strconv.Itoa(maxResults))
	if opts.Query != "" {
		values.Set("q", opts.Query)
	}
	if opts.PageToken != "" {
		values.Set("pageToken", opts.PageToken)
	}
	for _, labelID := range labelIDs {
		values.Add("labelIds", labelID)
	}

	endpoint := fmt.Sprintf("%s/gmail/v1/users/%s/messages?%s", c.baseURL, url.PathEscape(userID), values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build list messages request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("list messages request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", c.requestStatusError("list messages request failed", resp)
	}

	var payload struct {
		Messages      []domain.MessageRef `json:"messages"`
		NextPageToken string              `json:"nextPageToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, "", fmt.Errorf("decode list messages response: %w", err)
	}

	return payload.Messages, payload.NextPageToken, nil
}

func (c *Client) requestStatusError(prefix string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyText := strings.TrimSpace(string(body))
	if len(bodyText) > 1000 {
		bodyText = bodyText[:1000]
	}

	wwwAuthenticate := strings.TrimSpace(resp.Header.Get("WWW-Authenticate"))

	if bodyText != "" && wwwAuthenticate != "" {
		return fmt.Errorf("%s with status %d (www-authenticate=%q, body=%q)", prefix, resp.StatusCode, wwwAuthenticate, bodyText)
	}

	if bodyText != "" {
		return fmt.Errorf("%s with status %d (body=%q)", prefix, resp.StatusCode, bodyText)
	}

	if wwwAuthenticate != "" {
		return fmt.Errorf("%s with status %d (www-authenticate=%q)", prefix, resp.StatusCode, wwwAuthenticate)
	}

	return fmt.Errorf("%s with status %d", prefix, resp.StatusCode)
}

func (c *Client) GetMessage(ctx context.Context, userID, messageID string) (*domain.MailMessage, error) {
	if userID == "" {
		userID = "me"
	}

	endpoint := fmt.Sprintf("%s/gmail/v1/users/%s/messages/%s?format=full", c.baseURL, url.PathEscape(userID), url.PathEscape(messageID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build get message request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get message request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get message request failed with status %d", resp.StatusCode)
	}

	var payload gmailMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode get message response: %w", err)
	}

	attachments := extractAttachments(payload.Payload)
	headers := flattenHeaders(payload.Payload)

	msg := &domain.MailMessage{
		ID:            payload.ID,
		ThreadID:      payload.ThreadID,
		Subject:       headers["subject"],
		Sender:        headers["from"],
		Snippet:       payload.Snippet,
		PlainTextBody: extractPlainTextBody(payload.Payload),
		ReceivedAt:    parseRFC1123(headers["date"]),
		Attachments:   attachments,
	}

	if payload.InternalDate != "" {
		if ms, err := strconv.ParseInt(payload.InternalDate, 10, 64); err == nil {
			t := time.UnixMilli(ms).UTC()
			msg.InternalDate = &t
		}
	}

	return msg, nil
}

func (c *Client) DownloadAttachment(ctx context.Context, userID, messageID, attachmentID string) ([]byte, error) {
	if userID == "" {
		userID = "me"
	}

	endpoint := fmt.Sprintf("%s/gmail/v1/users/%s/messages/%s/attachments/%s", c.baseURL, url.PathEscape(userID), url.PathEscape(messageID), url.PathEscape(attachmentID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build download attachment request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download attachment request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download attachment request failed with status %d", resp.StatusCode)
	}

	var payload struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode attachment response: %w", err)
	}

	decoded, err := base64.URLEncoding.DecodeString(payload.Data)
	if err != nil {
		return nil, fmt.Errorf("decode attachment data: %w", err)
	}

	return decoded, nil
}

func (c *Client) DownloadMessageAttachments(ctx context.Context, userID, messageID string, refs []domain.MailAttachmentRef) ([]domain.DownloadedMailAttachment, error) {
	results := make([]domain.DownloadedMailAttachment, 0, len(refs))
	for _, ref := range refs {
		if ref.AttachmentID == "" {
			continue
		}

		data, err := c.DownloadAttachment(ctx, userID, messageID, ref.AttachmentID)
		if err != nil {
			return nil, err
		}

		results = append(results, domain.DownloadedMailAttachment{
			MailAttachmentRef: ref,
			Data:              data,
		})
	}

	return results, nil
}

func (c *Client) CreateLabel(ctx context.Context, userID, labelName string) (string, error) {
	if userID == "" {
		userID = "me"
	}

	payload := map[string]interface{}{
		"name":                  labelName,
		"labelListVisibility":   "labelShow",
		"messageListVisibility": "show",
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal label payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/gmail/v1/users/%s/labels", c.baseURL, url.PathEscape(userID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("build create label request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create label request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Attempt to read body for error details
		var errDetails string
		if b, _ := json.Marshal(resp.Body); len(b) > 0 {
			errDetails = string(b) // just roughly
		}
		return "", fmt.Errorf("create label request failed with status %d: %s", resp.StatusCode, errDetails)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode create label response: %w", err)
	}

	return result.ID, nil
}

func (c *Client) AddLabelToMessage(ctx context.Context, userID, messageID, labelID string) error {
	if userID == "" {
		userID = "me"
	}

	payload := map[string]interface{}{
		"addLabelIds": []string{labelID},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal modify message payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/gmail/v1/users/%s/messages/%s/modify", c.baseURL, url.PathEscape(userID), url.PathEscape(messageID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("build modify message request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("modify message request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("modify message request failed with status %d", resp.StatusCode)
	}

	return nil
}

type gmailMessageResponse struct {
	ID           string            `json:"id"`
	ThreadID     string            `json:"threadId"`
	Snippet      string            `json:"snippet"`
	InternalDate string            `json:"internalDate"`
	Payload      *gmailMessagePart `json:"payload"`
}

type gmailMessagePart struct {
	Filename string              `json:"filename"`
	MimeType string              `json:"mimeType"`
	Headers  []gmailHeader       `json:"headers"`
	Body     gmailPartBody       `json:"body"`
	Parts    []*gmailMessagePart `json:"parts"`
}

type gmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type gmailPartBody struct {
	AttachmentID string `json:"attachmentId"`
	Data         string `json:"data"`
	Size         int64  `json:"size"`
}

func extractAttachments(part *gmailMessagePart) []domain.MailAttachmentRef {
	if part == nil {
		return nil
	}

	refs := make([]domain.MailAttachmentRef, 0)
	var walk func(*gmailMessagePart)
	walk = func(node *gmailMessagePart) {
		if node == nil {
			return
		}

		if node.Body.AttachmentID != "" {
			filename := node.Filename
			if filename == "" {
				filename = path.Base(node.Body.AttachmentID)
			}

			refs = append(refs, domain.MailAttachmentRef{
				AttachmentID: node.Body.AttachmentID,
				Filename:     filename,
				MimeType:     node.MimeType,
				Size:         node.Body.Size,
			})
		}

		for _, child := range node.Parts {
			walk(child)
		}
	}

	walk(part)
	return refs
}

func flattenHeaders(part *gmailMessagePart) map[string]string {
	values := map[string]string{}
	if part == nil {
		return values
	}

	for _, header := range part.Headers {
		values[strings.ToLower(header.Name)] = header.Value
	}

	return values
}

func parseRFC1123(value string) *time.Time {
	if value == "" {
		return nil
	}

	t, err := time.Parse(time.RFC1123Z, value)
	if err != nil {
		t, err = time.Parse(time.RFC1123, value)
		if err != nil {
			return nil
		}
	}

	utc := t.UTC()
	return &utc
}

func extractPlainTextBody(part *gmailMessagePart) string {
	if part == nil {
		return ""
	}

	if strings.EqualFold(part.MimeType, "text/plain") && part.Body.AttachmentID == "" && part.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			return strings.TrimSpace(string(decoded))
		}
	}

	for _, child := range part.Parts {
		if content := extractPlainTextBody(child); content != "" {
			return content
		}
	}

	return ""
}
