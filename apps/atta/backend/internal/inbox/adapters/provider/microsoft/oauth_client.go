package microsoft

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
}

func NewOAuthHTTPClient(ctx context.Context, cfg OAuthConfig, credentialsJSON []byte) (*Client, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("microsoft oauth config is incomplete")
	}
	if len(credentialsJSON) == 0 {
		return nil, fmt.Errorf("microsoft oauth credentials are required")
	}

	var token oauth2.Token
	if err := json.Unmarshal(credentialsJSON, &token); err != nil {
		return nil, fmt.Errorf("decode microsoft oauth token: %w", err)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes: []string{
			"offline_access",
			"User.Read",
			"Mail.ReadWrite",
			"Mail.Send",
		},
		Endpoint: microsoft.AzureADEndpoint("common"),
	}

	return NewClient(oauthCfg.Client(ctx, &token)), nil
}
