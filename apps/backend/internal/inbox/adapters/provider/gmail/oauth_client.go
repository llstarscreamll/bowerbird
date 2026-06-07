package gmail

import (
	"context"
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
			"https://www.googleapis.com/auth/gmail.readonly",
		},
		Endpoint: google.Endpoint,
	}

	httpClient := oauthCfg.Client(ctx, &token)

	return NewClient(httpClient), nil
}
