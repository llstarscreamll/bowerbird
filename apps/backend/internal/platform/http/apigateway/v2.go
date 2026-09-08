package apigateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
)

func ProxyWithContext(h http.Handler) func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		httpReq, err := newHTTPRequest(ctx, req)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusInternalServerError}, err
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httpReq)
		return newV2Response(rec.Result()), nil
	}
}

func newHTTPRequest(ctx context.Context, req events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	method := req.RequestContext.HTTP.Method
	if method == "" {
		method = http.MethodGet
	}

	rawPath := req.RawPath
	if rawPath == "" {
		rawPath = req.RequestContext.HTTP.Path
	}
	if rawPath == "" {
		rawPath = "/"
	}

	rawURL := rawPath
	if req.RawQueryString != "" {
		rawURL += "?" + req.RawQueryString
	}

	body, err := decodeBody(req.Body, req.IsBase64Encoded)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.ContentLength = int64(len(body))
	httpReq.RemoteAddr = req.RequestContext.HTTP.SourceIP
	httpReq.Host = req.RequestContext.DomainName
	if httpReq.URL != nil && httpReq.URL.Host == "" {
		httpReq.URL.Host = req.RequestContext.DomainName
		httpReq.URL.Scheme = "https"
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}
	if len(req.Cookies) > 0 {
		httpReq.Header.Set("Cookie", strings.Join(req.Cookies, "; "))
	}

	return httpReq, nil
}

func decodeBody(body string, base64Encoded bool) ([]byte, error) {
	if body == "" {
		return nil, nil
	}
	if !base64Encoded {
		return []byte(body), nil
	}
	return base64.StdEncoding.DecodeString(body)
}

func newV2Response(res *http.Response) events.APIGatewayV2HTTPResponse {
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	headers := make(map[string]string, len(res.Header))
	var cookies []string
	for key, values := range res.Header {
		if strings.EqualFold(key, "Set-Cookie") {
			cookies = append(cookies, values...)
			continue
		}
		headers[key] = strings.Join(values, ",")
	}

	encoded, isB64 := encodeResponseBody(body, headers["Content-Type"])
	return events.APIGatewayV2HTTPResponse{
		StatusCode:      res.StatusCode,
		Headers:         headers,
		Cookies:         cookies,
		Body:            encoded,
		IsBase64Encoded: isB64,
	}
}

func encodeResponseBody(body []byte, contentType string) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	if isBinaryContent(contentType, body) {
		return base64.StdEncoding.EncodeToString(body), true
	}
	return string(body), false
}

func isBinaryContent(contentType string, body []byte) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch {
	case strings.HasPrefix(mediaType, "image/"),
		strings.HasPrefix(mediaType, "audio/"),
		strings.HasPrefix(mediaType, "video/"),
		mediaType == "application/octet-stream",
		mediaType == "application/pdf",
		mediaType == "application/zip",
		mediaType == "application/gzip":
		return true
	}
	return !utf8.Valid(body)
}
