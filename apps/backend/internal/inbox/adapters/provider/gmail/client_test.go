package gmail

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bowerbird/internal/inbox/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMessagesWithIncrementalQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/gmail/v1/users/me/messages", r.URL.Path)

		assert.Equal(t, "after:1716633600 -in:spam -in:sent", r.URL.Query().Get("q"))

		_, _ = w.Write([]byte(`{"messages":[{"id":"m1","threadId":"t1"}],"nextPageToken":"nxt"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	messages, nextPageToken, err := client.ListMessages(context.Background(), domain.ListMessagesOptions{
		UserID:     "me",
		Query:      "after:1716633600",
		MaxResults: 20,
	})

	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "m1", messages[0].ID)
	assert.Equal(t, "nxt", nextPageToken)
}

func TestListMessagesDefaultsToSpamExclusionQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "-in:spam -in:sent", r.URL.Query().Get("q"))
		_, _ = w.Write([]byte(`{"messages":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	_, _, err := client.ListMessages(context.Background(), domain.ListMessagesOptions{UserID: "me"})

	require.NoError(t, err)
}

func TestGetMessageExtractsHeadersAndAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/messages/"))

		_, _ = w.Write([]byte(`{
			"id":"m1",
			"threadId":"t1",
			"snippet":"Resumen del correo",
			"internalDate":"1716633600000",
			"payload":{
				"headers":[
					{"name":"Subject","value":"Factura Electronica"},
					{"name":"From","value":"proveedor@example.com"},
					{"name":"Date","value":"Tue, 25 May 2026 10:00:00 +0000"}
				],
				"parts":[
					{
						"mimeType":"text/plain",
						"body":{"data":"SG9sYSBlc3RlIGVzIGVsIGN1ZXJwbyBkZWwgY29ycmVvLg=="}
					},
					{
						"filename":"factura.xml",
						"mimeType":"application/xml",
						"body":{"attachmentId":"att-1","size":120}
					},
					{
						"filename":"factura.pdf",
						"mimeType":"application/pdf",
						"body":{"attachmentId":"att-2","size":220}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	msg, err := client.GetMessage(context.Background(), "me", "m1")

	require.NoError(t, err)
	assert.Equal(t, "Factura Electronica", msg.Subject)
	assert.Equal(t, "proveedor@example.com", msg.Sender)
	assert.Equal(t, "Resumen del correo", msg.Snippet)
	assert.Equal(t, "Hola este es el cuerpo del correo.", msg.PlainTextBody)
	require.Len(t, msg.Attachments, 2)
}

func TestGetMessageFromGoldenResponseWithAttachments(t *testing.T) {
	fixture := loadFixture(t, "gmail_message_with_attachments.golden.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/messages/"))
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	msg, err := client.GetMessage(context.Background(), "me", "18b9d18193dbd464")
	require.NoError(t, err)
	assert.Equal(t, "18b9d18193dbd464", msg.ID)
	assert.Equal(t, "18b9d18193dbd464", msg.ThreadID)
	assert.Equal(t, []string{"CATEGORY_PERSONAL"}, msg.LabelIDs)
	assert.Equal(t, "900277370; I SHOP COLOMBIA SAS; FETA19245; 01; I SHOP COLOMBIA SAS", msg.Subject)
	assert.Equal(t, "Documento Electronico ISHOP <dte_9002773704@dte.paperless.com.co>", msg.Sender)
	assert.Equal(t, "4821069", msg.HistoryID)
	assert.EqualValues(t, 74739, msg.SizeEstimate)
	require.NotEmpty(t, msg.HTMLBody)
	assert.Empty(t, msg.PlainTextBody)
	assert.NotNil(t, msg.Payload)
	assert.NotEmpty(t, msg.Headers)
	require.NotNil(t, msg.ReceivedAt)
	require.NotNil(t, msg.InternalDate)
	assert.True(t, msg.ReceivedAt.Equal(*msg.InternalDate))
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "fv90027737040532300457505.zip", msg.Attachments[0].Filename)
	assert.Equal(t, "application/octet-stream", msg.Attachments[0].MimeType)
	assert.EqualValues(t, 47070, msg.Attachments[0].Size)
	assert.Equal(t, "ANGjdJ-iVW0o1jsmB4iYkmyvu6yWy2CyaTI3TKcczOG0qXr3PCxhvBYeNAl2tTRFE10u5oGl7ejRhl7t0rwc-t8KJ-qoDJl_4nTxeMmcLww9slMOhdHUoPh-sETX4vf5auqcTYkpOQMcY4smS_eetlKduXjfAYwrBuJtdDymsPAz5bWViL6x2Jwofz8oC7HpUXZBPds3QS8Q6w8Nihs7QBJuVV8BH3FJyxDf3_PuFSnf8IZd7Qu-FFpNISWN_TpTHvwKj2quqDBfo-HUHFXZcPKcDp37OqNwjpT_p-1C9eclsbX4PTOUu3DIBMJI-hyTfo74olJZEx4iOpUG2qcxv61z9tV39Qo9Yo44YrUQbAPmvgngX8LGDwtHMCoRHwgLE6-j3DY5hnlBAirxksl1", msg.Attachments[0].AttachmentID)
}

func TestGetMessageFromGoldenResponseWithoutAttachments(t *testing.T) {
	fixture := loadFixture(t, "gmail_message_without_attachments.golden.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/messages/"))
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	msg, err := client.GetMessage(context.Background(), "me", "19e75618dba85c39")
	require.NoError(t, err)

	assert.Equal(t, "19e75618dba85c39", msg.ID)
	assert.Equal(t, "19e75618dba85c39", msg.ThreadID)
	assert.Equal(t, []string{"CATEGORY_PROMOTIONS", "UNREAD", "INBOX"}, msg.LabelIDs)
	assert.Equal(t, "Announcing a major product expansion for Neon", msg.Subject)
	assert.Equal(t, "Neon Changelog <changelog@neon.tech>", msg.Sender)
	assert.Equal(t, "43581", msg.HistoryID)
	assert.EqualValues(t, 19313, msg.SizeEstimate)
	require.NotNil(t, msg.Payload)
	assert.Equal(t, "multipart/alternative", msg.Payload.MimeType)
	require.Len(t, msg.Payload.Parts, 2)
	assert.Equal(t, "text/plain", msg.Payload.Parts[0].MimeType)
	assert.Equal(t, "text/html", msg.Payload.Parts[1].MimeType)
	require.NotNil(t, msg.InternalDate)
	require.NotNil(t, msg.ReceivedAt)
	require.NotEmpty(t, msg.PlainTextBody)
	require.NotEmpty(t, msg.HTMLBody)
	assert.Empty(t, msg.Attachments)
	assert.NotEmpty(t, msg.Headers)
}

func TestGetMessageExtractsPlainTextBodyFromUnpaddedBase64URL(t *testing.T) {
	bodyPayload := base64.RawURLEncoding.EncodeToString([]byte("Texto sin padding"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/messages/"))

		_, _ = w.Write([]byte(`{
			"id":"m2",
			"threadId":"t2",
			"snippet":"Snippet",
			"payload":{
				"headers":[
					{"name":"From","value":"proveedor@example.com"}
				],
				"parts":[
					{
						"mimeType":"text/plain",
						"body":{"data":"` + bodyPayload + `"}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	msg, err := client.GetMessage(context.Background(), "me", "m2")
	require.NoError(t, err)
	assert.Equal(t, "Texto sin padding", msg.PlainTextBody)
}

func TestGetMessageFallsBackToInternalDateWhenDateHeaderIsInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/messages/"))

		_, _ = w.Write([]byte(`{
			"id":"m3",
			"threadId":"t3",
			"snippet":"Snippet",
			"internalDate":"1699147682000",
			"payload":{
				"headers":[
					{"name":"Date","value":"not-a-date"}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	msg, err := client.GetMessage(context.Background(), "me", "m3")
	require.NoError(t, err)
	require.NotNil(t, msg.ReceivedAt)
	require.NotNil(t, msg.InternalDate)

	expected := time.UnixMilli(1699147682000).UTC()
	assert.True(t, msg.ReceivedAt.Equal(expected))
	assert.True(t, msg.InternalDate.Equal(expected))
}

func TestListMessagesStatusErrorIncludesResponseDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="https://accounts.google.com/", error="insufficient_scope"`)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"Request had insufficient authentication scopes.","status":"PERMISSION_DENIED"}}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	_, _, err := client.ListMessages(context.Background(), domain.ListMessagesOptions{UserID: "me"})
	require.Error(t, err)

	errText := err.Error()
	assert.Contains(t, errText, "status 403")
	assert.Contains(t, errText, "www-authenticate")
	assert.Contains(t, errText, "insufficient authentication scopes")
}

func TestDownloadAttachmentDecodesBase64URLData(t *testing.T) {
	payload := base64.URLEncoding.EncodeToString([]byte("xml-content"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/attachments/")

		_, _ = w.Write([]byte(`{"data":"` + payload + `"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	data, err := client.DownloadAttachment(context.Background(), "me", "m1", "att-1")
	require.NoError(t, err)
	assert.Equal(t, "xml-content", string(data))
}

func TestDownloadAttachmentFromGoldenResponse(t *testing.T) {
	fixture := loadFixture(t, "gmail_message_attachment.golden.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/attachments/")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	data, err := client.DownloadAttachment(context.Background(), "me", "18b9d18193dbd464", "ANGjdJ-iVW0o1jsm")
	require.NoError(t, err)
	require.Len(t, data, 47070)
	assert.Equal(t, []byte{'P', 'K', 0x03, 0x04}, data[:4])
}

func TestNewOAuthHTTPClientValidatesInputs(t *testing.T) {
	_, err := NewOAuthHTTPClient(context.Background(), OAuthConfig{}, []byte(`{"access_token":"a"}`))
	require.Error(t, err)

	_, err = NewOAuthHTTPClient(context.Background(), OAuthConfig{ClientID: "id", ClientSecret: "secret"}, nil)
	require.Error(t, err)

	_, err = NewOAuthHTTPClient(context.Background(), OAuthConfig{ClientID: "id", ClientSecret: "secret"}, []byte("not-json"))
	require.Error(t, err)
}

func loadFixture(t *testing.T, fileName string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", fileName))
	require.NoError(t, err)
	return string(body)
}
