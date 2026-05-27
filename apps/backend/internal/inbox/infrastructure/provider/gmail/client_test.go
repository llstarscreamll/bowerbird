package gmail

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

func TestListMessagesWithIncrementalQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if got := r.URL.Query().Get("q"); got != "after:1716633600" {
			t.Fatalf("expected incremental query, got %q", got)
		}

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
	if err != nil {
		t.Fatalf("list messages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].ID != "m1" || nextPageToken != "nxt" {
		t.Fatalf("unexpected list result: %#v, token=%s", messages[0], nextPageToken)
	}
}

func TestGetMessageExtractsHeadersAndAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/messages/") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

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
	if err != nil {
		t.Fatalf("get message failed: %v", err)
	}

	if msg.Subject != "Factura Electronica" {
		t.Fatalf("unexpected subject: %s", msg.Subject)
	}

	if msg.Sender != "proveedor@example.com" {
		t.Fatalf("unexpected sender: %s", msg.Sender)
	}

	if msg.Snippet != "Resumen del correo" {
		t.Fatalf("unexpected snippet: %s", msg.Snippet)
	}

	if msg.PlainTextBody != "Hola este es el cuerpo del correo." {
		t.Fatalf("unexpected plain text body: %s", msg.PlainTextBody)
	}

	if len(msg.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(msg.Attachments))
	}
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
	if err == nil {
		t.Fatal("expected list messages status error")
	}

	errText := err.Error()
	if !strings.Contains(errText, "status 403") {
		t.Fatalf("expected status in error, got %q", errText)
	}
	if !strings.Contains(errText, "www-authenticate") {
		t.Fatalf("expected WWW-Authenticate in error, got %q", errText)
	}
	if !strings.Contains(errText, "insufficient authentication scopes") {
		t.Fatalf("expected response body in error, got %q", errText)
	}
}

func TestDownloadAttachmentDecodesBase64URLData(t *testing.T) {
	payload := base64.URLEncoding.EncodeToString([]byte("xml-content"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/attachments/") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		_, _ = w.Write([]byte(`{"data":"` + payload + `"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetBaseURL(server.URL)

	data, err := client.DownloadAttachment(context.Background(), "me", "m1", "att-1")
	if err != nil {
		t.Fatalf("download attachment failed: %v", err)
	}

	if string(data) != "xml-content" {
		t.Fatalf("unexpected attachment data: %s", string(data))
	}
}

func TestNewOAuthHTTPClientValidatesInputs(t *testing.T) {
	if _, err := NewOAuthHTTPClient(context.Background(), OAuthConfig{}, []byte(`{"access_token":"a"}`)); err == nil {
		t.Fatal("expected oauth config validation error")
	}

	if _, err := NewOAuthHTTPClient(context.Background(), OAuthConfig{ClientID: "id", ClientSecret: "secret"}, nil); err == nil {
		t.Fatal("expected credentials validation error")
	}

	if _, err := NewOAuthHTTPClient(context.Background(), OAuthConfig{ClientID: "id", ClientSecret: "secret"}, []byte("not-json")); err == nil {
		t.Fatal("expected json decode error")
	}
}
