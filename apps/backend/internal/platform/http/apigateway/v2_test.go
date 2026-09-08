package apigateway

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyWithContextServesMux(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "acme", r.Header.Get("X-Tenant-ID"))
		assert.Equal(t, "q=1", r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := ProxyWithContext(mux)
	res, err := handler(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath:        "/health",
		RawQueryString: "q=1",
		Headers:        map[string]string{"x-tenant-id": "acme"},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
				Path:   "/health",
			},
			DomainName: "api.example.test",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, `{"status":"ok"}`, res.Body)
	assert.False(t, res.IsBase64Encoded)
	assert.Equal(t, "application/json", res.Headers["Content-Type"])
}

func TestProxyWithContextDecodesBase64Body(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		_, _ = w.Write(body)
	})

	payload := []byte("hello")
	res, err := ProxyWithContext(mux)(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath:         "/echo",
		Body:            base64.StdEncoding.EncodeToString(payload),
		IsBase64Encoded: true,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: http.MethodPost, Path: "/echo"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "hello", res.Body)
}

func TestProxyWithContextEncodesBinaryResponses(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /file", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF"))
	})

	res, err := ProxyWithContext(mux)(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/file",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: http.MethodGet, Path: "/file"},
		},
	})
	require.NoError(t, err)
	assert.True(t, res.IsBase64Encoded)
	decoded, err := base64.StdEncoding.DecodeString(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF"), decoded)
}
