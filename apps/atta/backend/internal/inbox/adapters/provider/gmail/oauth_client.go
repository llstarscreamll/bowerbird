package gmail

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
}

func NewOAuthHTTPClient(ctx context.Context, cfg OAuthConfig, credentialsJSON []byte) (*Client, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("gmail oauth config is incomplete")
	}

	if len(credentialsJSON) == 0 {
		return nil, fmt.Errorf("gmail oauth credentials are required")
	}

	var token oauth2.Token
	if err := json.Unmarshal(credentialsJSON, &token); err != nil {
		return nil, fmt.Errorf("decode gmail oauth token: %w", err)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes: []string{
			"email",
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.send",
		},
		Endpoint: google.Endpoint,
	}

	httpClient := oauthCfg.Client(ctx, &token)
	client := NewClient(httpClient)
	client.SetQuotaKey(quotaKeyFromToken(token))
	return client, nil
}

func quotaKeyFromToken(token oauth2.Token) string {
	raw := token.RefreshToken
	if raw == "" {
		raw = token.AccessToken
	}
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
